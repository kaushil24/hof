package main_trial

#AppConfig: {
    name:        string
    environment: string
    region:      string @runinject(tasks.deploy.input.region)
}