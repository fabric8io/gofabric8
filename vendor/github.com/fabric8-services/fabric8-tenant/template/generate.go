//go:generate sh -c "curl http://central.maven.org/maven2/io/fabric8/online/packages/fabric8-online-team/$TEAM_VERSION/fabric8-online-team-$TEAM_VERSION-openshift.yml > fabric8-online-team-openshift.yml"
//go:generate sh -c "curl http://central.maven.org/maven2/io/fabric8/online/packages/fabric8-online-jenkins/$TEAM_VERSION/fabric8-online-jenkins-$TEAM_VERSION-openshift.yml > fabric8-online-jenkins-openshift.yml"
//go:generate sh -c "curl http://central.maven.org/maven2/io/fabric8/online/packages/fabric8-online-jenkins-quotas-oso/$TEAM_VERSION/fabric8-online-jenkins-quotas-oso-$TEAM_VERSION-openshift.yml > fabric8-online-jenkins-quotas-oso-openshift.yml"
//go:generate sh -c "curl http://central.maven.org/maven2/io/fabric8/online/packages/fabric8-online-che/$TEAM_VERSION/fabric8-online-che-$TEAM_VERSION-openshift.yml > fabric8-online-che-openshift.yml"
//go:generate sh -c "curl http://central.maven.org/maven2/io/fabric8/online/packages/fabric8-online-che-quotas-oso/$TEAM_VERSION/fabric8-online-che-quotas-oso-$TEAM_VERSION-openshift.yml > fabric8-online-che-quotas-oso-openshift.yml"

//go:generate sh -c "curl http://central.maven.org/maven2/io/fabric8/online/packages/fabric8-online-team/$TEAM_VERSION/fabric8-online-team-$TEAM_VERSION-k8s-template.yml > fabric8-online-team-kubernetes.yml"
//go:generate sh -c "curl http://central.maven.org/maven2/io/fabric8/online/packages/fabric8-online-jenkins/$TEAM_VERSION/fabric8-online-jenkins-$TEAM_VERSION-k8s-template.yml > fabric8-online-jenkins-kubernetes.yml"
//go:generate sh -c "curl http://central.maven.org/maven2/io/fabric8/online/packages/fabric8-online-che/$TEAM_VERSION/fabric8-online-che-$TEAM_VERSION-k8s-template.yml > fabric8-online-che-kubernetes.yml"

//go:generate sh -c "curl http://central.maven.org/maven2/io/fabric8/platform/apps/exposecontroller/$PLATFORM_VERSION/exposecontroller-$PLATFORM_VERSION-k8s-template.yml > fabric8-online-expose-kubernetes.yml"
package template
