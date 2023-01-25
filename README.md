# Themis #

This README would normally document whatever steps are necessary to get your application up and running.

### What is this repository (Themis) for? ###

Creating, reading and managing rancher clusters. Here is an auth/sys overview.

![Overview of System](themis.png)

### Starting commands ###
* brew install go
* go install github.com/swaggo/swag/cmd/swag@latest

### To find paths ### 
* echo $PATH
* go env | grep GOPATH

### PATH zshr file ###
"# If you come from bash you might have to change your $PATH.
export PATH=$HOME/bin:/usr/local/bin:/Users/"URE NAME"/go/bin:$PATH "

### Then these commands ### 
* go mod download
* swag init -g cmd/main.go  

### Step 5 ### 
* setup .env file(import from one of admins)
* go install github.com/joho/godotenv/cmd/godotenv@latest
* docker-compose up -d 
* go run cmd/main.go

### Swagger docs generation ###
* swag init -g cmd/main.go

Link to swagger:
* http://localhost:9000/swagger/index.html 


```
{
    "name":"Adam",
    "email":"adam3@test.com",
    "password":"12345"
}
```

### For dotenv file ###
ask admin for permission. 

### How to build with Docker ###
Project is using standard golang container from docker hub, check: https://hub.docker.com/_/golang 

```
$ docker build -t themis .
$ docker run -it --rm themis
```
### Copied Code ###

Some code was copied from https://github.com/dgrijalva/jwt-go/blob/master/hmac_example_test.go which is under MIT license.

### Contribution guidelines ###

* Writing tests
* Code review
* Other guidelines

### Who do I talk to? ###

* Repo owner or admin
* Other community or team contact