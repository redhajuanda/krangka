package main

import "gitlab.sicepat.tech/pka/sds/cmd"

// @title dsd Service API
// @description This is a documentation for dsd Service RESTful APIs. <br>

// @securityDefinitions.basic BasicAuth
// @in header
// @name Authorization

// @securityDefinitions.apikey BearerToken
// @in header
// @name Authorization

// @BasePath /pka/sds

func main() {

	cmd.Run()

}