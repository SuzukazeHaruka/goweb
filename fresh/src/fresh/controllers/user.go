package controllers

import (
	"encoding/base64"
	"fmt"
	"fresh/models"
	"github.com/astaxie/beego"
	"github.com/astaxie/beego/orm"
	"github.com/astaxie/beego/utils"
	"github.com/gomodule/redigo/redis"
	"regexp"
	"strconv"
	"time"
)

type UserController struct {
	beego.Controller
}


//处理提交的新地址
func (this*UserController)HandleCenterSite()  {
	username := this.GetSession("userName")
	//根据用户名获取用户信息
	//根据用户名获取用户信息
	o := orm.NewOrm()
	var user models.User
	user.Name=username.(string)
	o.Read(&user,"Name")
	name:=this.GetString("receiverName")
	zipCode:=this.GetString("zipCode")
	phone:=this.GetString("phone")
	address:=this.GetString("address")
	//数据校验
	if name == ""||address==""||zipCode==""||address=="" || phone==""{
		this.Data["errmsg"] = "添加地址数据不能为空"
		this.TplName = "user_center_site.html"
		return
	}
	var addr models.Address

	addr.Isdefault=true
	//把原来的默认地址改为false
	err := o.Read(&addr, "Isdefault")
	if err==nil{
		addr.Isdefault=false
		o.Update(&addr)
	}
	addr.Receiver=name
	addr.User=&user
	addr.Addr=address
	addr.Phone=phone
	addr.Zipcode=zipCode
	addr.Isdefault=true
	_, err = o.Insert(&addr)

	if err != nil {
		this.Data["errmsg"]="插入地址错误请重新插入"
		this.TplName="user_center_site.html"
		return
	}
	this.Redirect("/user/userCenterSite",302)
}



//-------------------用户中心地址页

func (this*UserController)ShowUserCenterSite()  {
	username := this.GetSession("userName")
	//根据用户名获取用户信息
	o := orm.NewOrm()
	user := models.User{Name:username.(string)}
	o.Read(&user,"Name")
	var addr models.Address
	o.QueryTable("Address").RelatedSel("User").Filter("User__id",user.Id).
		Filter("Isdefault",true).One(&addr)
	this.Data["addr"] = addr
	this.Data["userName"] = username
	this.Layout = "userCenterLayout.html"//这里我们也套用了视图布局
	this.TplName = "user_center_site.html"
}

//-------------用户中心订单页
//展示用户中心订单页
func(this*UserController)ShowUserCenterOrder(){

	userName := this.GetSession("userName")
	if userName == nil{
		this.Data["userName"] = ""

	}else{
		this.Data["userName"] = userName.(string)
	}

	o := orm.NewOrm()

	var user models.User
	user.Name = userName.(string)
	o.Read(&user,"Name")




	//获取订单表的数据
	var orderInfos []models.OrderInfo
	o.QueryTable("OrderInfo").RelatedSel("User").Filter("User__Id",user.Id).All(&orderInfos)

	goodsBuffer := make([]map[string]interface{},len(orderInfos))

	for index,orderInfo := range orderInfos{

		var orderGoods []models.OrderGoods
		o.QueryTable("OrderGoods").RelatedSel("OrderInfo","GoodsSKU").Filter("OrderInfo__Id",orderInfo.Id).All(&orderGoods)

		temp := make(map[string]interface{})
		temp["orderInfo"] = orderInfo
		temp["orderGoods"] = orderGoods

		goodsBuffer[index] = temp

	}

	this.Data["goodsBuffer"] = goodsBuffer

	//订单商品表

	this.Layout = "userCenterLayout.html"
	this.TplName = "user_center_order.html"
}





//----------------------用户信息
func (this*UserController)ShowUserCenterInfo()  {
	userName := this.GetSession("userName")
	//根据用户名获取用户信息

	o := orm.NewOrm()
	var user models.User
	user.Name=userName.(string)
	o.Read(&user,"Name")
	//高级查询表关联
	var addr models.Address
	o.QueryTable("Address").RelatedSel("User").Filter("User__Name",userName).Filter("Isdefault",true).One(&addr)
	if addr.Id == 0{
		this.Data["addr"] = ""

	}else{
		this.Data["addr"] = addr
	}

	var goods[] models.GoodsSKU
	conn, err := redis.Dial("tcp", "112.74.172.1:6379")
	if err != nil {
		fmt.Println("redis连接错误",err)
		return
	}
	reply, err := conn.Do("lrange", "history"+strconv.Itoa(user.Id), 0, 4)
	beego.Error(err)
	replyInts, _ := redis.Ints(reply, err)
	for _, val := range replyInts {
		var temp models.GoodsSKU
		o.QueryTable("GoodsSKU").Filter("Id",val).One(&temp)
		goods = append(goods, temp)
	}
	this.Data["goods"]=goods
	this.Data["userName"]=userName
	this.Layout="userCenterLayout.html"
	this.TplName = "user_center_info.html"
}






//----用户退出
func(this*UserController)Logout(){
	this.DelSession("username")
	this.Redirect("/",302)
}


//-----用户登陆
func (this*UserController)HandleLogin()  {
	var user models.User
	this.ParseForm(&user)
	checked := this.GetString("check")
	if user.Name==""||user.PassWord==""{
		this.Data["errmsg"]="账户或者密码错误"
		this.TplName="login.html"
		return
	}
	o := orm.NewOrm()
	err := o.Read(&user,"Name","PassWord")
	if err != nil {
		this.Data["errmsg"] = "用户名或密码错误,请重新登陆！"
		this.TplName = "login.html"
		return
	}
	if !user.Active{
		this.Data["errmsg"] = "该用户没有激活，请县激活！"
		this.TplName = "login.html"
		return
	}
	if checked=="on"{
		temp := base64.StdEncoding.EncodeToString([]byte(user.Name))
		this.Ctx.SetCookie("username",temp,time.Second*3600)
	}else {
		this.Ctx.SetCookie("username",user.Name,-1)
	}
	beego.Info(user.Name)
	this.SetSession("userName",user.Name)

	this.Redirect("/",302)

}













func(this*UserController)ShowLogin(){
	username:=this.Ctx.GetCookie("username")
	temp, _ := base64.StdEncoding.DecodeString(username)
	if username!=""{
		this.Data["username"]=string(temp)
		this.Data["checked"]="checked"

	}else{
		this.Data["username"]=""
		this.Data["cheked"]=""
	}
	this.TplName="login.html"
}


func (this*UserController)ShowReg(){
	this.TplName="register.html"
}








func (this*UserController)ActiveUser(){
	//获取传输过来的数据
	id, err := this.GetInt("id")
	if err!=nil{
		this.Data["errmsg"]="激活路径不正确,请重新确定之后登陆"
		this.TplName="login.html"
		return
	}
	//更新激活字段
	o := orm.NewOrm()
	user := models.User{Id: id}
	err = o.Read(&user)
	if err!=nil{
		this.Data["errmsg"]="激活路径不正确,请重新确定之后登陆"
		this.TplName="login.html"
		return
	}
	user.Active=true
	_,err = o.Update(&user)
	if err != nil{
		this.Data["errmsg"] = "激活失败，请重新确定之后登陆！"
		this.TplName = "login.html"
	}
	this.Redirect("/login",302)
}




/**
	处理注册
 */
func (this*UserController)HandleReg()  {
	user := models.User{}
	this.ParseForm(&user)

	VerifyReg(this,&user)
	o := orm.NewOrm()
	err := o.Read(&user, "Name")
	if err != orm.ErrNoRows {
		beego.Error(err)
		this.Data["errmsg"]="用户名已存在"
		this.TplName="register.html"
		return
	}
	_, err = o.Insert(&user)
	if err != nil {
		this.Data["errmsg"]="插入失败,请重新注册"
		this.TplName="register.html"
		return
	}

	config:=`{"username":"294080257@qq.com","password":"kxsnqivklkwjcbcj","host":"smtp.qq.com","port":587}`
	temail := utils.NewEMail(config)
	temail.To=[]string{user.Email}//指定收件人的邮箱,这个邮箱就是用户注册时的邮箱
	temail.From="294080257@qq.com"//指定发件人的邮箱地址,这里我们使用的qq邮箱
	temail.Subject="凉风青葉用户激活"//指定邮件的标题
	////指定邮件的内容。该内容发送到用户的邮箱中以后，
	// 该用户打开邮箱，可以将该URL地址复制到地址栏中，
	// 敲回车键，就会向该指定的URL地址发送请求，
	// 我们在该地址对应的方法中，接收该用户的ID,然后根据该Id,
	// 查询出用户的信息后，将其对应的一个属性，Active设置为true,
	// 表明用户已经激活了，那么用户就可以登录了。
	temail.HTML="复制该连接到浏览器中激活：http://127.0.0.1:8080/active?id="+strconv.Itoa(user.Id)

	//发送邮件
	temail.Send()
	if err != nil {
		this.Data["errmsg"]="发送激活邮件失败,请重新注册"
		this.TplName="register.html"
	}
	this.Ctx.WriteString("注册成功,请前往邮箱激活")
	//this.Redirect("/",302)

}


//验证前端数据
func VerifyReg(this *UserController,user *models.User)  {
	//对前端发送过来的数据进行非空校验
	if user.Name=="" || user.PassWord=="" || user.Email==""{
		this.Data["errmsg"]="输入信息不完整,请重新输入"
		this.TplName="register.html"
		return
	}
	//判断密码和确认密码是否正确
	if user.PassWord!=this.GetString("cpwd"){
		this.Data["errmsg"]="两次输入的密码不一致"
		this.TplName="register.html"
		this.TplName="register.html"
		return
	}
	//对邮箱格式的校验:注意这里使用的是regexp下的Compile方法
	reg,_:=regexp.Compile("^[A-Za-z0-9\u4e00-\u9fa5]+@[a-zA-Z0-9_-]+(\\.[a-zA-Z0-9_-]+)+$")
	res := reg.FindString(user.Email)
	if res==""{
		this.Data["errmsg"]="邮箱格式不正确"
		this.TplName="register.html"
		return
	}




}