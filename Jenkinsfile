pipeline {
  agent any
  stages {
    stage('node-a1') {
      parallel {
        stage('node-a1') {
          steps {
            sh 'sleep 10s'
          }
        }
        stage('node-d1') {
          steps {
            sh 'sleep 5s'
          }
        }
        stage('node-t1') {
          steps {
            sh 'sleep 30s'
          }
        }
      }
    }
    stage('everything is ready') {
      steps {
        echo 'its done'
      }
    }
  }
}