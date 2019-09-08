package main

import (
	_ "fresh/models"
	_ "fresh/routers"
	"github.com/astaxie/beego"
)

func main() {
	beego.Run()
}

