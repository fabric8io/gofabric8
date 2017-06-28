package cmds

import (
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"k8s.io/kubernetes/pkg/client/restclient"
	k8client "k8s.io/kubernetes/pkg/client/unversioned"

	"github.com/spf13/cobra"

	"strings"
)

// NB(chmou): I don't like this either :\
var CONFIG_MAP_LIST_JSON = `{"kind":"ConfigMapList","apiVersion":"v1","metadata":{"selfLink":"/api/v1/namespaces/test/configmaps","resourceVersion":"174694"},"items":[{"metadata":{"name":"fabric8-environments","namespace":"developer","selfLink":"/api/v1/namespaces/developer/configmaps/fabric8-environments","uid":"f1db4293-5b3d-11e7-b9a0-fa163e96266f","resourceVersion":"146433","creationTimestamp":"2017-06-27T13:38:46Z","labels":{"group":"io.fabric8.online.packages","kind":"environments","project":"fabric8-online-team","provider":"fabric8","version":"1.0.175"},"annotations":{"description":"Defines the environments used by your Continuous Delivery pipelines.","fabric8.console/iconUrl":"https://cdn.rawgit.com/fabric8io/fabric8-console/master/app-kubernetes/src/main/fabric8/icon.svg"}},"data":{"key":"name: key\nnamespace: developer\norder: 5\n","key2":"name: key2\nnamespace: developer\norder: 3\n"}},{"metadata":{"name":"fabric8-pipelines","namespace":"developer","selfLink":"/api/v1/namespaces/developer/configmaps/fabric8-pipelines","uid":"bccefdda-5820-11e7-b9a0-fa163e96266f","resourceVersion":"39403","creationTimestamp":"2017-06-23T14:32:08Z","labels":{"group":"io.fabric8.online.packages","project":"fabric8-online-team","provider":"fabric8","version":"1.0.175"}},"data":{"cd-branch-patterns":"- master","ci-branch-patterns":"- PR-.*","disable-itests-cd":"false","disable-itests-ci":"false"}}]}`
var CONFIG_MAP_PUT_JSON = `{"kind":"ConfigMap","apiVersion":"v1","metadata":{"name":"foo","namespace":"developer","selfLink":"/api/v1/namespaces/test/configmaps/foo","uid":"665aad53-5c16-11e7-b9a0-fa163e96266f","resourceVersion":"175503","creationTimestamp":"2017-06-28T15:28:13Z"}}`

func TestCreateEnvironArgs(t *testing.T) {
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
