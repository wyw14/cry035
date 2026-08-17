package catalog

import (
	"context"
	"time"

	"github.com/wyw14/cry035/internal/domain/equipment"
	"github.com/wyw14/cry035/internal/domain/maintenance"
)

type Repository interface {
	ListBuildings(context.Context) ([]equipment.Building, error)
	ListEquipment(context.Context) ([]equipment.Equipment, error)
	GetEquipment(context.Context, string) (equipment.Equipment, error)
	SaveEquipment(context.Context, equipment.Equipment) error
	ListPrograms(context.Context) ([]maintenance.Program, error)
	GetProgram(context.Context, string) (maintenance.Program, error)
	SaveProgram(context.Context, maintenance.Program) error
}

type IDGenerator interface{ New() string }
type Clock interface{ Now() time.Time }

type Service struct {
	repo  Repository
	ids   IDGenerator
	clock Clock
}

func New(repo Repository, ids IDGenerator, clock Clock) *Service {
	return &Service{repo: repo, ids: ids, clock: clock}
}

func (s *Service) Buildings(ctx context.Context) ([]equipment.Building, error) {
	return s.repo.ListBuildings(ctx)
}

func (s *Service) Equipment(ctx context.Context) ([]equipment.Equipment, error) {
	return s.repo.ListEquipment(ctx)
}

func (s *Service) EquipmentByID(ctx context.Context, id string) (equipment.Equipment, error) {
	return s.repo.GetEquipment(ctx, id)
}

type CreateEquipment struct {
	Code            string
	Name            string
	BuildingID      string
	ModelID         string
	Location        string
	ResponsibleUnit string
}

func (s *Service) CreateEquipment(ctx context.Context, input CreateEquipment) (equipment.Equipment, error) {
	now := s.clock.Now().UTC()
	item := equipment.Equipment{
		ID: s.ids.New(), Code: input.Code, Name: input.Name, BuildingID: input.BuildingID,
		ModelID: input.ModelID, Location: input.Location, ResponsibleUnit: input.ResponsibleUnit,
		Status: equipment.StatusRunning, Version: 1, CreatedAt: now, UpdatedAt: now,
	}
	if err := s.repo.SaveEquipment(ctx, item); err != nil {
		return equipment.Equipment{}, err
	}
	return item, nil
}

func (s *Service) Programs(ctx context.Context) ([]maintenance.Program, error) {
	return s.repo.ListPrograms(ctx)
}

func (s *Service) ProgramByID(ctx context.Context, id string) (maintenance.Program, error) {
	return s.repo.GetProgram(ctx, id)
}
