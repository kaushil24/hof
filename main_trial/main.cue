package main_trial

// import "example.com/myproject/config"

tasks: {
    setup: {
        @flow(step1)
        output: {
            region: string @runinject(REGION)
        }
    }

    deploy: {
        @flow(step2)
        dep: setup
        input: {
            app_name: string @runinject(APP_NAME)
            region:   string @runinject(setup.region)
        }
        config: config.#AppConfig & {
            name:        input.app_name
            environment: string @runinject(ENVIRONMENT)
        }
        output: {
            deployment_id: "dep-123"
        }
    }
}