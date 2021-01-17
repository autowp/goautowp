package goautowp

import (
	"github.com/gin-gonic/gin"
	"net/http"
)

// RecaptchaController Main Object
type RecaptchaController struct {
	config RecaptchaConfig
}

// NewRecaptchaController constructor
func NewRecaptchaController(config RecaptchaConfig) (*RecaptchaController, error) {

	s := &RecaptchaController{
		config: config,
	}

	return s, nil
}

func (s *RecaptchaController) SetupRouter(apiGroup *gin.RouterGroup) {
	apiGroup.GET("/recaptcha", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"publicKey": s.config.PublicKey,
		})
	})
}
