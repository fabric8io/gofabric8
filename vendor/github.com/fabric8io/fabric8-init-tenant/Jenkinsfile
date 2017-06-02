#!/usr/bin/groovy
@Library('github.com/fabric8io/fabric8-pipeline-library@master')
def utils = new io.fabric8.Utils()
goTemplate{
  dockerNode{
    ws {
      checkout scm

      if (utils.isCI()){
        echo 'CI is provided by ci.centos'

      } else if (utils.isCD()){
        def v = goRelease{
          githubOrganisation = 'fabric8io'
          dockerOrganisation = 'fabric8'
          project = 'fabric8-init-tenant'
          dockerBuildOptions = '--file Dockerfile.deploy'
        }

        pushPomPropertyChangePR{
            propertyName = 'init-tenant.version'
            projects = [
                    'fabric8io/fabric8-platform'
            ]
            version = v
            containerName = 'go'
        }
      }
    }
  }
}
