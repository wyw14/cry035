package httptransport

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/wyw14/cry035/internal/application/catalog"
	"github.com/wyw14/cry035/internal/middleware"
)

func (s *Server) registerCatalog(api *gin.RouterGroup) {
	api.GET("/buildings", s.listBuildings)
	api.GET("/equipment", s.listEquipment)
	api.GET("/equipment/:id", s.getEquipment)
	api.POST("/equipment", middleware.RequireRole("safety_manager", "planner"), s.createEquipment)
	api.GET("/programs", s.listPrograms)
	api.GET("/programs/:id", s.getProgram)
}

func (s *Server) listBuildings(c *gin.Context) {
	items, err := s.services.Catalog.Buildings(c.Request.Context())
	if err != nil {
		s.error(c, err)
		return
	}
	respondPage(c, items)
}

func (s *Server) listEquipment(c *gin.Context) {
	items, err := s.services.Catalog.Equipment(c.Request.Context())
	if err != nil {
		s.error(c, err)
		return
	}
	respondPage(c, items)
}

func (s *Server) getEquipment(c *gin.Context) {
	item, err := s.services.Catalog.EquipmentByID(c.Request.Context(), c.Param("id"))
	if err != nil {
		s.error(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"item": item, "request_id": middleware.GetRequestID(c)})
}

type createEquipmentRequest struct {
	Code            string `json:"code" validate:"required,max=64"`
	Name            string `json:"name" validate:"required,max=128"`
	BuildingID      string `json:"building_id" validate:"required"`
	ModelID         string `json:"model_id" validate:"required"`
	Location        string `json:"location" validate:"required,max=128"`
	ResponsibleUnit string `json:"responsible_unit" validate:"required,max=128"`
}

func (s *Server) createEquipment(c *gin.Context) {
	var request createEquipmentRequest
	if !s.bind(c, &request) {
		return
	}
	item, err := s.services.Catalog.CreateEquipment(c.Request.Context(), catalog.CreateEquipment{
		Code: request.Code, Name: request.Name, BuildingID: request.BuildingID, ModelID: request.ModelID,
		Location: request.Location, ResponsibleUnit: request.ResponsibleUnit,
	})
	if err != nil {
		s.error(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"item": item, "request_id": middleware.GetRequestID(c)})
}

func (s *Server) listPrograms(c *gin.Context) {
	items, err := s.services.Catalog.Programs(c.Request.Context())
	if err != nil {
		s.error(c, err)
		return
	}
	respondPage(c, items)
}

func (s *Server) getProgram(c *gin.Context) {
	item, err := s.services.Catalog.ProgramByID(c.Request.Context(), c.Param("id"))
	if err != nil {
		s.error(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"item": item, "request_id": middleware.GetRequestID(c)})
}
