/**
 * Copyright (C) 2015 Red Hat, Inc.
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
	"net/http"
	"net/http/httptest"

	"k8s.io/kubernetes/pkg/client/restclient"
	k8client "k8s.io/kubernetes/pkg/client/unversioned"
)

// NB(chmou): I don't like this either :\
var CONFIG_MAP_LIST_JSON = `{"kind": "ConfigMapList",
 "apiVersion": "v1",
 "items": [{
   "data": {
     "key": "name: key\nnamespace: developer\norder: 5\n",
     "key2": "name: key2\nnamespace: developer\norder: 3\n"
   }
 }]}`

var CONFIG_MAP_PUT_JSON = `{"kind":"ConfigMap","apiVersion":"v1","metadata":{"name":"foo","namespace":"developer","selfLink":"/api/v1/namespaces/test/configmaps/foo","uid":"665aad53-5c16-11e7-b9a0-fa163e96266f","resourceVersion":"175503","creationTimestamp":"2017-06-28T15:28:13Z"}}`

func fakeTestRestResponder(url, jsonresp string) (*httptest.Server, *k8client.Client) {
	mux := http.NewServeMux()
	mux.Handle(url, http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		io.WriteString(w, jsonresp)
	}))
	server := httptest.NewServer(mux)
	c, _ := k8client.New(&restclient.Config{
		Host: server.URL,
	})
	return server, c
}
