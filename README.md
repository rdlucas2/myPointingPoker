### work locally
```
docker build -t mypointingpoker:local --target local .
docker run -it --rm -v "$(pwd):/working" -p 8080:8080 --name mypointingpoker mypointingpoker:local

#inside the container
go run *.go
```

### test
```
docker build -t mypointingpoker:test --target test .
docker run -it --rm -v "$(pwd)/out:/out" --name mypointingpoker mypointingpoker:test
```

### sonar scan
```
docker run -it --rm -e SONAR_HOST_URL="http://host.docker.internal:9000" -e SONAR_LOGIN="<your-generated-token>" -v "$(pwd):/usr/src" sonarsource/sonar-scanner-cli
```

### run
```
docker build -t mypointingpoker:run --target run .
docker run -it --rm -p 8080:8080 --name mypointingpoker mypointingpoker:run
```

### artifact
```
docker build -t mypointingpoker:latest --target artifact .
docker run -it --rm -p 8080:8080 --name mypointingpoker mypointingpoker:latest
```

### trivy scan
```
docker run -it --rm -v /var/run/docker.sock:/var/run/docker.sock -v "$(pwd)/out:/out" aquasec/trivy image --format table --output /out/trivy-report.txt --scanners vuln mypointingpoker:latest
```

### deploy to AWS
```
cd terraform
#configure aws credentials
terraform init
terraform validate
terraform plan
terraform apply
```

### infracost example
```
cd terraform
infracost auth login
infracost breakdown --path .
```

### Other notes:
- infracost: https://www.infracost.io/docs/

### TODO:
- listen for client disconnect and remove user from db
- write unit tests, fix sonarqube issues
- add volume mounts for docker run commands and parameterize db file from env var
- fix db/ui oddness when not deployed locally, not refreshing all the time properly
- improve CSS
- add checkov to terraform / output to sonarqube as well
- refactor db so it's coming from single source... dynamodb? postgres? is there a way to continue using sqlite across multiple instances?