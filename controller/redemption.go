package controller

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"one-api/common"
	"one-api/model"
	"os"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/stripe/stripe-go/v76"
)

func GetAllRedemptions(c *gin.Context) {
	p, _ := strconv.Atoi(c.Query("p"))
	if p < 0 {
		p = 0
	}
	redemptions, err := model.GetAllRedemptions(p*common.ItemsPerPage, common.ItemsPerPage)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    redemptions,
	})
	return
}

//处理用户
func StripeCallback(c *gin.Context) {
	//首先验证是不是stripe服务器发来的消息。
	var StripeWebHookSecret = os.Getenv("Stripe_Webhook_Secret")
	payload, ioerr := ioutil.ReadAll(c.Request.Body)
	sigHeader := c.GetHeader("Stripe-Signature")
	if !model.IsStripeWebhookValid(payload, sigHeader, StripeWebHookSecret) {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "the request seems not from stripe",
		})
		return
	}
	if ioerr != nil {
		fmt.Println("读文件错误")
		c.JSON(http.StatusOK, gin.H{
			"sucess":  false,
			"message": ioerr.Error(),
		})
		return
	}
	event := stripe.Event{}
	if err := json.Unmarshal(payload, &event); err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": err.Error(),
		})

		return
	}
	// err := json.NewDecoder(c.Request.Body).Decode(&event)
	// if err != nil {
	// 	fmt.Println("解析event错误")
	// 	c.JSON(http.StatusOK, gin.H{
	// 		"success": false,
	// 		"message": err.Error(),
	// 	})
	// 	return
	// }
	switch event.Type {
	case "payment_intent.succeeded":
		/* 	var paymentIntent stripe.PaymentIntent
		err := json.Unmarshal(event.Data.Raw, &paymentIntent)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error parsing webhook JSON:%v\n", err)
			c.JSON(http.StatusOK, gin.H{
				"success": false,
				"message": err.Error(),
			})
			return
		}
		//处理付款逻辑
		amount := paymentIntent.AmountReceived
		//model.ProcessStripPaymentIntentSucceeded(amount, userId)
		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"message": paymentIntent,
			"amount":  amount,
		}) */
		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"message": "no need to handle",
		})
		return

	case "checkout.session.completed":
		var session stripe.CheckoutSession
		err := json.Unmarshal(event.Data.Raw, &session)
		//处理付款逻辑
		if err != nil {
			c.JSON(http.StatusOK, gin.H{
				"success": false,
				"message": err.Error(),
			})
			return
		}
		amount := session.AmountTotal
		email := session.CustomerDetails.Email
		fmt.Println(email)
		userId := model.GetUserIdByEmail(email)
		fmt.Println(userId)
		model.ProcessStripPaymentIntentSucceeded(amount, userId)
		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"message": "",
			"amount":  amount,
			"email":   email,
		})
		return
	default:
		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"message": "no need to handle",
		})
	}

}
func SearchRedemptions(c *gin.Context) {
	keyword := c.Query("keyword")
	redemptions, err := model.SearchRedemptions(keyword)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    redemptions,
	})
	return
}

func GetRedemption(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}
	redemption, err := model.GetRedemptionById(id)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    redemption,
	})
	return
}

func AddRedemption(c *gin.Context) {
	redemption := model.Redemption{}
	err := c.ShouldBindJSON(&redemption)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}
	if len(redemption.Name) == 0 || len(redemption.Name) > 20 {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "兑换码名称长度必须在1-20之间",
		})
		return
	}
	if redemption.Count <= 0 {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "兑换码个数必须大于0",
		})
		return
	}
	if redemption.Count > 100 {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "一次兑换码批量生成的个数不能大于 100",
		})
		return
	}
	var keys []string
	for i := 0; i < redemption.Count; i++ {
		key := common.GetUUID()
		cleanRedemption := model.Redemption{
			UserId:      c.GetInt("id"),
			Name:        redemption.Name,
			Key:         key,
			CreatedTime: common.GetTimestamp(),
			Quota:       redemption.Quota,
		}
		err = cleanRedemption.Insert()
		if err != nil {
			c.JSON(http.StatusOK, gin.H{
				"success": false,
				"message": err.Error(),
				"data":    keys,
			})
			return
		}
		keys = append(keys, key)
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    keys,
	})
	return
}

func DeleteRedemption(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	err := model.DeleteRedemptionById(id)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
	})
	return
}

func UpdateRedemption(c *gin.Context) {
	statusOnly := c.Query("status_only")
	redemption := model.Redemption{}
	err := c.ShouldBindJSON(&redemption)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}
	cleanRedemption, err := model.GetRedemptionById(redemption.Id)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}
	if statusOnly != "" {
		cleanRedemption.Status = redemption.Status
	} else {
		// If you add more fields, please also update redemption.Update()
		cleanRedemption.Name = redemption.Name
		cleanRedemption.Quota = redemption.Quota
	}
	err = cleanRedemption.Update()
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    cleanRedemption,
	})
	return
}
