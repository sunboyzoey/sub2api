package routes

import (
	"github.com/Wei-Shaw/sub2api/internal/handler"

	"github.com/gin-gonic/gin"
)

func RegisterSalesRoutes(v1 *gin.RouterGroup, h *handler.Handlers) {
	sales := v1.Group("/sales")
	{
		sales.GET("/catalog", h.Sales.ListStoreCatalog)

		partner := sales.Group("/partner")
		{
			partner.GET("/packages", h.Sales.ListPartnerPackages)
			partner.POST("/orders/provision", h.Sales.ProvisionOrder)
			partner.GET("/orders/:external_order_id", h.Sales.GetPartnerOrder)
		}
	}
}
