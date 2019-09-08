package controllers

import (
	"fmt"
	"fresh/models"
	"github.com/astaxie/beego"
	"github.com/astaxie/beego/orm"
	"github.com/gomodule/redigo/redis"
	"math"
	"strconv"
)

type GoodsController struct {
	beego.Controller
}

func(this*GoodsController)HandleSearch(){
	//处理搜索
		//获取数据
		goodsName := this.GetString("goodsName")
		o := orm.NewOrm()
		var goods []models.GoodsSKU
		//校验数据
		if goodsName == ""{
			o.QueryTable("GoodsSKU").All(&goods)
			this.Data["goods"] = goods
			ShowLaout(&this.Controller)
			this.TplName = "search.html"
			return
		}

		//处理数据
		o.QueryTable("GoodsSKU").Filter("Name__icontains",goodsName).All(&goods)

		//返回视图
		this.Data["goods"] = goods
		ShowLaout(&this.Controller)
		this.TplName = "search.html"

}
func ShowLaout(this*beego.Controller){
	//查询类型
	o := orm.NewOrm()
	var types []models.GoodsType
	o.QueryTable("GoodsType").All(&types)
	this.Data["types"] = types
	//获取用户信息
	GetUser(this)
	//指定layout
	this.Layout = "goodLayout.html"
}

func (this*GoodsController)ShowGoodsList()  {
	typeId, err := this.GetInt("typeId")
	if err != nil {
		beego.Info("获取类型ID错误")
		this.Redirect("/",302)
		return
	}
	//获取商品类型
	var types []models.GoodsType
	o := orm.NewOrm()
	o.QueryTable("GoodsType").All(&types)
	this.Data["types"]=types


	//获取新品数据
	//获取两个新品
	var goodsNew []models.GoodsSKU
	o.QueryTable("GoodsSKU").RelatedSel("GoodsType").Filter("GoodsType__Id",typeId).OrderBy("Time").Limit(2,0).All(&goodsNew)
	this.Data["goodsNew"] = goodsNew
	GetUser(&this.Controller)
	page:=5
	count,_ := o.QueryTable("GoodsSKU").RelatedSel("GoodsType").Filter("GoodsType__Id",typeId).Count()

	pageCount := math.Ceil(float64(count)/float64(page))

	pageIndex,err := this.GetInt("pageIndex")
	if err != nil{
		pageIndex = 1
	}

	pages := PageTool(int(pageCount),pageIndex)

	//获取当前类型的商品
	var goodsSKUs []models.GoodsSKU
	//根据不同的选项获取不同的数据
	sort := this.GetString("sort")
	//如果sort等于空，就按照默认排序
	if sort == ""{	o.QueryTable("GoodsSKU").RelatedSel("GoodsType").Filter("GoodsType__Id",typeId).Limit(page,pageIndex).All(&goodsSKUs)

	}else if sort == "price"{	o.QueryTable("GoodsSKU").RelatedSel("GoodsType").Filter("GoodsType__Id",typeId).OrderBy("Price").Limit(page,pageIndex).All(&goodsSKUs)

	}else{
		o.QueryTable("GoodsSKU").RelatedSel("GoodsType").Filter("GoodsType__Id",typeId).OrderBy("Sales").Limit(page,pageIndex).All(&goodsSKUs)
	}
	this.Data["sort"]=sort
	this.Data["goods"]=goodsSKUs
	//获取上一页页码
	prePage := pageIndex - 1
	if prePage <= 1{
		prePage = 1
	}
	this.Data["pagePre"] = prePage

	//获取下一页页码
	nextPage := pageIndex + 1
	if nextPage > int(pageCount){
		nextPage = int(pageCount)
	}
	this.Data["pageNext"] = nextPage

	this.Data["pages"] = pages
	this.Data["typeId"] = typeId


	this.Layout="goodLayout.html"
	this.TplName="list.html"
}




func PageTool(pageCount int,pageIndex int)[]int{

	var pages []int
	if pageCount <= 5{
		pages = make([]int,pageCount)
		for i,_ := range pages{
			pages[i] = i + 1
		}

		//pages = [1,2,..,pageCount]
	}else if pageIndex <= 3{
		//pages := make([]int,5)
		pages = []int{1,2,3,4,5}
	}else if pageIndex > pageCount - 3 {
		//pages = [6, 7, 8, 9, 10]
		pages = []int{pageCount -4,pageCount - 3,pageCount - 2,pageCount -1 ,pageCount}
	}else {
		pages = []int{pageIndex - 2,pageIndex -1 ,pageIndex,pageIndex + 1, pageIndex + 2}
	}
	return pages

}


func (this*GoodsController)ShowDetail()  {
	id, err := this.GetInt("id")
	if err != nil {
		beego.Error("获取数据不存在")
		this.Redirect("/",302)
		return
	}
	o := orm.NewOrm()
	var goodsTypes [] models.GoodsType
	o.QueryTable("GoodsType").All(&goodsTypes)
	this.Data["types"]=goodsTypes
	//获取商品详情
	var goods models.GoodsSKU
	o.QueryTable("GoodsSKU").RelatedSel("GoodsType","Goods").Filter("Id",id).One(&goods)


	//获取新品数据
	//获取两个新品
	var goodsNew []models.GoodsSKU

	o.QueryTable("GoodsSKU").RelatedSel("GoodsType").Filter("GoodsType__Id",goods.GoodsType.Id).OrderBy("Time").Limit(2,0).All(&goodsNew)


	this.Data["goodsNew"] = goodsNew
	//添加用户历史浏览记录,需要查询是否登录,只有登录之后可以添加历史浏览记录
	userName := this.GetSession("userName")
	if userName!=nil{
		//查询用户信息
		var user models.User
		user.Name=userName.(string)
		o.Read(&user,"Name")
		conn, err := redis.Dial("tcp", "112.74.172.1:6379")
		if err != nil {
			fmt.Println("连接redis失败",err)
			return
		}
		//出入历史记录
		reply, err := conn.Do("lpush", "history"+strconv.Itoa(user.Id), id)
		reply, _ = redis.Bool(reply, err)
		if reply==false{
			beego.Info("插入浏览数据错误")
		}
	}
	GetUser(&this.Controller)
	this.Data["goods"]=goods
	this.Layout="goodLayout.html"
	this.TplName="detail.html"
}



func GetUser(this *beego.Controller){
	userName := this.GetSession("userName")
	if userName == nil{
		this.Data["userName"] = ""

	}else{
		this.Data["userName"] = userName.(string)
	}


}

//展示首页
func(this*GoodsController)ShowIndex(){
	GetUser(&this.Controller)
	o := orm.NewOrm()
	//获取类型数据
	var goodsTypes []models.GoodsType
	o.QueryTable("GoodsType").All(&goodsTypes)
	this.Data["goodsTypes"] = goodsTypes

	//获取轮播图数据
	var indexGoodsBanner []models.IndexGoodsBanner
	o.QueryTable("IndexGoodsBanner").OrderBy("Index").All(&indexGoodsBanner)
	this.Data["indexGoodsBanner"] = indexGoodsBanner

	//获取促销商品数据
	var promotionGoods []models.IndexPromotionBanner
	o.QueryTable("IndexPromotionBanner").OrderBy("Index").All(&promotionGoods)
	this.Data["promotionsGoods"] = promotionGoods

	//首页展示商品数据
	goods := make([]map[string]interface{},len(goodsTypes))

	//向切片interface中插入类型数据
	for index, value := range goodsTypes{
		//获取对应类型的首页展示商品
		temp := make(map[string]interface{})
		temp["type"] = value
		goods[index] = temp
	}
	//商品数据


	for _,value := range goods{
		var textGoods []models.IndexTypeGoodsBanner
		var imgGoods []models.IndexTypeGoodsBanner
		//获取文字商品数据
		o.QueryTable("IndexTypeGoodsBanner").RelatedSel("GoodsType","GoodsSKU").OrderBy("Index").Filter("GoodsType",value["type"]).Filter("DisplayType",0).All(&textGoods)
		//获取图片商品数据
		o.QueryTable("IndexTypeGoodsBanner").RelatedSel("GoodsType","GoodsSKU").OrderBy("Index").Filter("GoodsType",value["type"]).Filter("DisplayType",1).All(&imgGoods)

		value["textGoods"] = textGoods
		value["imgGoods"] = imgGoods
	}
	this.Data["goods"] = goods


	this.Layout="layout.html"
	this.TplName = "index.html"
}