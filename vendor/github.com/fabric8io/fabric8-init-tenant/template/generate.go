//go:generate sh -c "curl -v http://central.maven.org/maven2/io/fabric8/online/packages/fabric8-online-team/$TEAM_VERSION/fabric8-online-team-$TEAM_VERSION-openshift.yml > fabric8-online-team-openshift.yml"
//go:generate sh -c "curl -v http://central.maven.org/maven2/io/fabric8/online/packages/fabric8-online-jenkins/$TEAM_VERSION/fabric8-online-jenkins-$TEAM_VERSION-openshift.yml > fabric8-online-jenkins-openshift.yml"
//go:generate sh -c "curl -v http://central.maven.org/maven2/io/fabric8/online/packages/fabric8-online-che/$TEAM_VERSION/fabric8-online-che-$TEAM_VERSION-openshift.yml > fabric8-online-che-openshift.yml"
package template
