package controllers

import (
	"fresh/models"
	"github.com/astaxie/beego"
	"github.com/smartwalle/alipay"
	"strconv"
	"github.com/astaxie/beego/orm"

	"github.com/gomodule/redigo/redis"
	"time"
	"strings"
	"fmt"

)

type OrderController struct {
	beego.Controller
}

func(this*OrderController)ShowOrder(){
	//获取数据
	skuids :=this.GetStrings("skuid")
	beego.Info(skuids)
	//校验数据
	if len(skuids) == 0{
		beego.Info("请求数据错误")
		this.Redirect("/user/cart",302)
		return
	}

	//处理数据
	o := orm.NewOrm()
	conn,_ := redis.Dial("tcp","112.74.172.1:6379")
	defer conn.Close()
	//获取用户数据
	var user models.User
	userName := this.GetSession("userName")
	user.Name = userName.(string)
	o.Read(&user,"Name")

	goodsBuffer := make([]map[string]interface{},len(skuids))

	totalPrice := 0
	totalCount := 0
	for index,skuid := range skuids{
		temp := make(map[string]interface{})

		id ,_ := strconv.Atoi(skuid)
		//查询商品数据
		var goodsSku models.GoodsSKU
		goodsSku.Id = id
		o.Read(&goodsSku)

		temp["goods"] = goodsSku
		//获取商品数量
		count,_ :=redis.Int(conn.Do("hget","cart_"+strconv.Itoa(user.Id),id))
		temp["count"] = count
		//计算小计
		amount := goodsSku.Price * count
		temp["amount"] = amount

		//计算总金额和总件数
		totalCount += count
		totalPrice += amount

		goodsBuffer[index] = temp
	}

	this.Data["goodsBuffer"] = goodsBuffer

	//获取地址数据
	var addrs []models.Address
	o.QueryTable("Address").RelatedSel("User").Filter("User__Id",user.Id).All(&addrs)
	this.Data["addrs"] = addrs

	//传递总金额和总件数
	this.Data["totalPrice"] = totalPrice
	this.Data["totalCount"] = totalCount
	transferPrice := 10
	this.Data["transferPrice"] = transferPrice
	this.Data["realyPrice"] = totalPrice + transferPrice

	//传递所有商品的id
	this.Data["skuids"] = skuids

	//返回视图
	this.TplName = "place_order.html"
}

//添加订单
func(this*OrderController)AddOrder(){
	//获取数据
	addrid,_ :=this.GetInt("addrid")
	payId,_ :=this.GetInt("payId")
	skuid := this.GetString("skuids")
	ids := skuid[1:len(skuid)-1]

	skuids := strings.Split(ids," ")


	beego.Error(skuids)
	//totalPrice,_ := this.GetInt("totalPrice")
	totalCount,_ := this.GetInt("totalCount")
	transferPrice,_ :=this.GetInt("transferPrice")
	realyPrice,_:=this.GetInt("realyPrice")


	resp := make(map[string]interface{})
	defer this.ServeJSON()
	//校验数据
	if len(skuids) == 0{
		resp["code"] = 1
		resp["errmsg"] = "数据库链接错误"
		this.Data["json"] = resp
		return
	}
	//处理数据
	//向订单表中插入数据
	o := orm.NewOrm()

	o.Begin()//标识事务的开始

	userName := this.GetSession("userName")
	var user models.User
	user.Name = userName.(string)
	o.Read(&user,"Name")

	var order models.OrderInfo
	order.OrderId = time.Now().Format("2006010215030405")+strconv.Itoa(user.Id)
	order.User = &user
	order.Orderstatus = 1
	order.PayMethod = payId
	order.TotalCount = totalCount
	order.TotalPrice = realyPrice
	order.TransitPrice = transferPrice
	//查询地址
	var addr models.Address
	addr.Id = addrid
	o.Read(&addr)

	order.Address = &addr

	//执行插入操作
	o.Insert(&order)


	//想订单商品表中插入数据
	conn,_ :=redis.Dial("tcp","112.74.172.1:6379")

	for _,skuid := range skuids{
		id,_ := strconv.Atoi(skuid)

		var goods models.GoodsSKU
		goods.Id = id
		i := 3

		for i> 0{
		o.Read(&goods)

		var orderGoods models.OrderGoods

		orderGoods.GoodsSKU = &goods
		orderGoods.OrderInfo = &order

		count ,_ :=redis.Int(conn.Do("hget","cart_"+strconv.Itoa(user.Id),id))

		if count > goods.Stock{
			resp["code"] = 2
			resp["errmsg"] = "商品库存不足"
			this.Data["json"] = resp
			o.Rollback()  //标识事务的回滚
			return
		}

		preCount := goods.Stock

		time.Sleep(time.Second * 5)
		beego.Info(preCount,user.Id)

		orderGoods.Count = count

		orderGoods.Price = count * goods.Price

		o.Insert(&orderGoods)

		goods.Stock -= count
		goods.Sales += count

		updateCount,_:=o.QueryTable("GoodsSKU").Filter("Id",goods.Id).Filter("Stock",preCount).Update(orm.Params{"Stock":goods.Stock,"Sales":goods.Sales})
		if updateCount == 0{
			if i >0 {
				i -= 1
				continue
			}
			resp["code"] = 3
			resp["errmsg"] = "商品库存改变,订单提交失败"
			this.Data["json"] = resp
			o.Rollback()  //标识事务的回滚
			return
		}else{
			conn.Do("hdel","cart_"+strconv.Itoa(user.Id),goods.Id)
			break
		}
		}

	}

	//返回数据
	o.Commit()  //提交事务
	resp["code"] = 5
	resp["errmsg"] = "ok"
	this.Data["json"] = resp

}

//处理支付
func(this*OrderController)HandlePay(){
	var aliPublicKey = "MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAh8C04Eqb+KzswdWh78kyX7I8sJidup5/NQxIEzg00aW28w94sJ0Jbc5pdp6cIGDftoYeSGjBllH7mab7PXUdaUwFSKDDx/ST8guKG0df8Ilo/xhY8yudVyIuFr7H8G8hLX+urErN3SAL2m2yksVdjHVcyQnkE/X3qQ/TiUildabW66gLIFAbgM+ayDUtao1BVuR/zK28Z/4vnz0fOu6wnNwNPoNdcoLI3pX6+VRG/0LWXZ+zcwZKCNJErJrhXJPiZarAnGploE3TjT/e4jnKDkBEGnth44TLdJDWCdI2c0BDWMf01P5Gh4K4K8EaxCZB3CpEuqvQ67TGB4nLBRyjSQIDAQAB"



	var privateKey = "MIIEpQIBAAKCAQEAx07zhUqQfHvIGK86l2UTyX8Eeaj/864gJYEwlxGq38jx/RqaDhHPP87/fzpPOkVoftLml9zknLN1WSBql50TglkwOZhcD41RGtuhSTOQfnbYpLekhLdsafoYSqkTMPtsoQpt3xeI6uF1BPOJYLzCH6FVJNjDS1s/QMSlXk+X+nqUX58/krC7aW7XwAqpbLWQA0Lth1g5GxhS2R70LjbfzUVt2lwrMwcer2xeb0GKHz3aykWKTNFMkoP5das0q9b8AUveczl6YEM4ujPv4R6aAQiS1hHorwKLpvTyktPuvf0dNQbKAmPHAgJh5ZKwkvkpWrEQRdefMZgSJf4PuK48kQIDAQABAoIBAQCeRihE1W3gLTw9vgm9aFtKTD/1jSuVC9YjcnBvx2v2wtDIunNUcPgwJ+Xl1xxLngrZjAnq11QEzM6HtKJxPB/eB42wbznMb+DUf02ZoAVDKIXqaJuReUfy8NSRlarT3xXo3StbWok0XU5cXDngRIW0MJ444JpWIWQdvwvD1VlPYtnmRpvD5MlcZLMszaGcJHE7EttXeGvLLGhChLA7hEVM7lHIaeOeF8ma2SUpN/ee7C4B3whDFzwrM08SMe1E6Ivpz33xcO2mziT+R5UMC7RxuJMST2lHCmku3JNGzNekd/UfXS3VXKZsch1JMbysrjO1cH4CYLTKFjNU8FfwmFexAoGBAOJzxD/2RMcMGp7Im0PJEimcuRGH0cwpL9K54shfzb3tuy3DVC1ye8/8ENszKO/Q3DRpJkyM4tHpiJMfe8cDV3SexGHdiOyJTseDxecgL0pCI33ZWy02cKCCDTMgEyv++ujxyxpwmxtKQCLNxpG/uSuFOVoXpt+dbPwL3NJemqT3AoGBAOFQfnQtT39T9AfFHtpJl6WZ5DzKew6A+AwKePmc2F/dg+vOQKguyjE7sj4uZXOmIsgX518b9QPdvCuyH+0zmJx8OoRF+C2MXYz0BYpCuz9POwBufGVgvoq1HoOtOntlTyrhdBcNCicFD9GuFbfFmb/+uftioeMymgfITOVHbDC3AoGAI2dB+VYBLrVfvA9U5uYaptLPxEPdsvOFfIZ/RCBmRBlUuDTfhjNt0/hukjaPYd7fbno5+KHWHEdMiOPVMCn/lEX2Ie7Gp2RYIq0hVZ8chZmNfvFqZckrFoz+j02mcaxtgdm7jSiptzyGhmpxbvvwcTNk4gbsme08yrL4FROhTcUCgYEAkUutIBIQD9X9qf0N1kpaxmmk6ybPkBzO2ETwmlbwmXFpnuiUfWAe9vy+Bqc4uQlLqKjxhT2sFOAqdisZt4bsRQ0/Vwkf749yzHCYGf7KbRsUu0SEZ4OpnB0MHnHZIrXEBaz5hdvczijPeLHAQ4/jhBIpsNh7+N0qwxYBsGEMfaUCgYEAswfrSExYzjZha65a/mRhr4A0/BFdZbWXCLDyk2iLImtLFxkTttSET285t2qshLuJfOe8fCqyhnrYlFExySwlSFA7mjZHpHevOS8F12ORvG8pC9MV7zRQfwSc5IGkeUpLEVUszwVMqG1sov0EG2J/VuLCghPFZyL1LpRH7yW+GHM="


	var appId = "2016101300679066"
	var client, err1 = alipay.New(appId, aliPublicKey, privateKey, false)
	if err1 != nil {
		beego.Error("创建支付宝客户端错误:",err1)
	}

	//获取数据
	orderId := this.GetString("orderId")
	totalPrice := this.GetString("totalPrice")

	var p = alipay.TradePagePay{}
	p.NotifyURL = "http://xxx"
	p.ReturnURL = "http://localhost:8080/user/payok"
	p.Subject = "天天生鲜购物平台"
	p.OutTradeNo = orderId
	p.TotalAmount = totalPrice
	p.ProductCode = "FAST_INSTANT_TRADE_PAY"

	var url, err = client.TradePagePay(p)
	if err != nil {
		fmt.Println(err)
	}

	var payURL = url.String()
	this.Redirect(payURL,302)
}

//支付成功
func(this*OrderController)PayOk(){
	//获取数据
	//out_trade_no=999998888777
	orderId := this.GetString("out_trade_no")


	//校验数据
	if orderId ==""{
		beego.Info("支付返回数据错误")
		this.Redirect("/user/userCenterOrder",302)
		return
	}

	//操作数据

	o := orm.NewOrm()
	count,_:=o.QueryTable("OrderInfo").Filter("OrderId",orderId).Update(orm.Params{"Orderstatus":2})
	if count == 0{
		beego.Info("更新数据失败")
		this.Redirect("/user/userCenterOrder",302)
		return
	}

	//返回视图
	this.Redirect("/user/userCenterOrder",302)
}
