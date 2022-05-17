
def executeAPIRun() {
        // This block is required due to discremency in env names test / platform-test so on
        env.test_env=env.TARGET_ENV

        if ( env.TARGET_ENV.contains("test") ) { 
            env.test_env="Test" }
        if ( env.TARGET_ENV.contains("stage") ) {
            env.test_env="Stage" }
        if ( env.TARGET_ENV.contains("prod") ) {
            env.test_env="Prod" }

       def jobBuildContract = build job: 'test-automation/AutomationTestRun', parameters: [
              string(name: 'PRODUCT', value: "PlaceHolder"),string(name: 'PROJECT', value: "PlaceHolder"),string(name: 'env', value: "${test_env}"),string(name: 'Message', value: "Executed as part of ${env.SERVICE} microservices ")
              ],propagate: false

        def jobResultContract = jobBuildContract.getResult()

        if (jobResultContract == 'SUCCESS') {
            def jobBuildApi = build job: 'test-automation/AutomationTestRun', parameters: [
            string(name: 'PRODUCT', value: "AssessTestAutomation"),string(name: 'PROJECT', value: "AssessAPIAutomation"),string(name: 'env', value: "${test_env}"),string(name: 'Message', value: "Executed as part of ${env.SERVICE} microservices")
            ], propagate: false
            
            def jobResultApi = jobBuildApi.getResult()

            if (jobResultApi != 'SUCCESS') {
                println("AutomationTestRun failed for AssessAPIAutomation with result: ${jobResultApi}")
            }
            else{
                println("AutomationTestRun testRun is success for AssessAPIAutomation.")
            }
        } 
        else{
            println("AutomationTestRun failed for AssessAPIContractAutomation with result: ${jobResultContract}")
        }
}

return this
