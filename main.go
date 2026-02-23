package main

import "github.com/redhajuanda/krangka/cmd"

// @title Krangka Service API
// @description This is a documentation for Krangka Service RESTful APIs. <br>

// @securityDefinitions.basic BasicAuth
// @in header
// @name Authorization

// @securityDefinitions.apikey BearerToken
// @in header
// @name Authorization

// @BasePath /platform/krangka

func main() {

	cmd.Run()

}
