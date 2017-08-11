/**
 * Copyright (C) 2017 Red Hat, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *         http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */
package cmds

import (
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	restclient "k8s.io/kubernetes/pkg/client/restclient"
	k8client "k8s.io/kubernetes/pkg/client/unversioned"

	"github.com/spf13/cobra"

	"strings"
)

func TestGetEnviron(t *testing.T) {
	cmd := &cobra.Command{}
	detectedNS := "test"
	var args = []string{}

	server, c := fakeTestRestResponder("/", CONFIG_MAP_LIST_JSON)
	defer server.Close()

	err := getEnviron(cmd, args, detectedNS, c)
	if err != nil {
		t.Fatal(err)
	}
}

func TestCreateEnviron(t *testing.T) {
	cmd := &cobra.Command{}
	detectedNS := "test"
	ch := make(chan string, 1)

	mux := http.NewServeMux()
	mux.Handle("/", http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		body, _ := ioutil.ReadAll(req.Body)
		ch <- string(body)
		io.WriteString(w, CONFIG_MAP_PUT_JSON)
	}))

	mux.Handle("/api/v1/namespaces/"+detectedNS+"/configmaps", http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		io.WriteString(w, CONFIG_MAP_LIST_JSON)
	}))
	server := httptest.NewServer(mux)

	defer server.Close()

	c, _ := k8client.New(&restclient.Config{
		Host: server.URL,
	})

	badargs := []string{"hello", "moto"}
	err := createEnviron(cmd, badargs, detectedNS, c)
	if err == nil {
		t.Fatalf("bad argument passing has falled %v should not be allowed", badargs)
	} else {
		if !strings.ContainsAny("you are missing an assignment like foo=bar", err.Error()) {
			t.Fatalf("Invalid error catch in the test: %v", err)
		}
	}

	missingargs := []string{"name=foo"}
	err = createEnviron(cmd, missingargs, detectedNS, c)
	if err == nil {
		t.Fatalf("not all arguments has been detected properly")
	}

	badorder := []string{"order=foo"}
	err = createEnviron(cmd, badorder, detectedNS, c)
	if err == nil {
		t.Fatalf("order integer should have been passed properly")
	} else {
		if !strings.ContainsAny("Cannot use", err.Error()) {
			t.Fatalf("Invalid error catch in the test: %v", err)
		}
	}

	goodargs := []string{"name=GOOD", "namespace=VERYGOOD", "order=1"}
	err = createEnviron(cmd, goodargs, detectedNS, c)

	if err != nil {
		t.Fatalf("Error while creating the configmaps: %+v", err)
	}

	bodyinserted := <-ch
	if !strings.ContainsAny("name=good", bodyinserted) || !strings.ContainsAny("namespace=VERYGOOD", bodyinserted) {
		t.Fatalf("Bad requested JSON: %s", bodyinserted)
	}

}
