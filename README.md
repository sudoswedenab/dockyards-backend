# Themis #

This README would normally document whatever steps are necessary to get your application up and running.

### What is this repository (Themis) for? ###

Creating, reading and managing rancher clusters. Here is an auth/sys overview.

![Overview of System](themis.png)

### Starting commands ###
* go mod download
* docker-compose up -d 
* go run cmd/main.go

Swagger docs generation
* swag init -g cmd/main.go

```
{
    "name":"Adam",
    "email":"adam3@test.com",
    "password":"12345"
}
```

### For dotenv file ###
ask admin for permission. 

### How do I get set up? ###
* Summary of set up
* Configuration
* Dependencies
* Database configuration
* How to run tests
* Deployment instructions

### Copied Code ###

Some code was copied from https://github.com/dgrijalva/jwt-go/blob/master/hmac_example_test.go which is under MIT license.

### Contribution guidelines ###

* Writing tests
* Code review
* Other guidelines

### Who do I talk to? ###

* Repo owner or admin
* Other community or team contact