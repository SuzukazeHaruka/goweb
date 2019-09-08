package routers

import (
	"fresh/controllers"
	"fresh/models"
	"github.com/astaxie/beego"
	"github.com/astaxie/beego/context"
	"github.com/astaxie/beego/orm"
	"github.com/gomodule/redigo/redis"
	"strconv"
)

func init() {
	beego.InsertFilter("/user/*",beego.BeforeExec,filterFunc)
	beego.InsertFilter("/cart/*",beego.BeforeExec,filterCartFunc)
	beego.Router("/", &controllers.GoodsController{},"get:ShowIndex")
    beego.Router("/register",&controllers.UserController{},"get:ShowReg;post:HandleReg")
	beego.Router("/active",&controllers.UserController{},"get:ActiveUser")
	beego.Router("/login",&controllers.UserController{},"get:ShowLogin;post:HandleLogin")
	beego.Router("/user/logout",&controllers.UserController{},"get:ShowLogin")
	beego.Router("/user/userCenterInfo",&controllers.UserController{},"get:ShowUserCenterInfo")
	beego.Router("/user/userCenterOrder",&controllers.UserController{},"get:ShowUserCenterOrder")
	beego.Router("/user/userCenterSite",&controllers.UserController{},"get:ShowUserCenterSite;post:HandleCenterSite")
	beego.Router("/goodsDetail",&controllers.GoodsController{},"get:ShowDetail")
	beego.Router("/goodsList",&controllers.GoodsController{},"get:ShowGoodsList")
	beego.Router("/goodsSearch",&controllers.GoodsController{},"post:HandleSearch")
	beego.Router("/user/addCart",&controllers.CartController{},"post:HandleAddCart")
	beego.Router("/user/cart",&controllers.CartController{},"get:ShowCart")
	//更新购物车数量
	beego.Router("/user/UpdateCart",&controllers.CartController{},"post:HandleUpdateCart")
	//删除购物车数据
	beego.Router("/user/deleteCart",&controllers.CartController{},"post:DeleteCart")

	//展示订单页面
	beego.Router("/user/showOrder",&controllers.OrderController{},"post:ShowOrder")
	//添加订单
	beego.Router("/user/addOrder",&controllers.OrderController{},"post:AddOrder")
	//处理支付
	beego.Router("/user/pay",&controllers.OrderController{},"get:HandlePay")
	//支付成功
	beego.Router("/user/payok",&controllers.OrderController{},"get:PayOk")
}

var filterFunc = func(ctx *context.Context){
	userName := ctx.Input.Session("userName")
	if userName == nil{
		ctx.Redirect(302,"/login")
		return
	}
}

var filterCartFunc= func(ctx *context.Context) {
	resp := make(map[string]interface{})
	defer ctx.Input.IsAjax()
	//处理数据
	//购物车数据存在redis中，用hash
	conn,err :=redis.Dial("tcp","112.74.172.1:6379")
	if err != nil{
		beego.Info("redis数据库链接错误")
		return
	}
	userName := ctx.Input.Session("userName")
	o := orm.NewOrm()
	var user models.User
	user.Name=userName.(string)
	o.Read(&user)
	rep,err :=conn.Do("hlen","cart_"+strconv.Itoa(user.Id))
	//回复助手函数
	cartCount ,_ :=redis.Int(rep,err)

	resp["code"] = 5
	resp["msg"] = "Ok"
	resp["cartCount"] = cartCount
}
