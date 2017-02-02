#!/usr/bin/groovy
@Library('github.com/fabric8io/fabric8-pipeline-library@master')
def dummy
goTemplate{
  dockerNode{

    if (env.BRANCH_NAME.startsWith('PR-')){
      echo 'Running CI pipeline'
      goMake{
        githubOrganisation = 'fabric8io'
        dockerOrganisation = 'fabric8'
        project = 'gofabric8'
      }
    } else if (env.BRANCH_NAME.equals('master')){
      echo 'Running CD pipeline'
      def v = goRelease{
        githubOrganisation = 'fabric8io'
        dockerOrganisation = 'fabric8'
        project = 'gofabric8'
      }

      updateDownstreamDependencies(v)
    }
  }
}

def updateDownstreamDependencies(v) {
  pushPomPropertyChangePR {
    propertyName = 'gofabric8.version'
    projects = [
            'fabric8io/fabric8-devops'
    ]
    version = v
  }
}
