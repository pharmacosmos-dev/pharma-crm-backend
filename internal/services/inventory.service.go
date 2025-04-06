package services

import (
	"github.com/pharma-crm-backend/domain"
	"github.com/pharma-crm-backend/pkg/utils"
)

func (s *Services) CreateInventory(req *domain.InventoryRequest) error {
	req.PublicId = utils.GenerateCode()
	var id string
	err := s.db.Raw(`
	INSERT INTO inventories (public_id, store_id, name, type)
	VALUES (?, ?, ?, ?) RETURNING id`,
		req.PublicId, req.StoreId, req.Name, req.Type,
	).Scan(&id).Error
	if err != nil {
		s.log.Error("ERROR on creating inventory: ", err)
		return err
	}
	if len(req.Products) > 0 {
		for _, product := range req.Products {
			err = s.db.Exec(`
			INSERT INTO inventory_details (
				inventory_id, product_id
			) VALUES (?, ?)
			`, id, product.ProductId).Error
			if err != nil {
				s.log.Error("ERROR on creating inventory: ", err)
				return err
			}
		}
	}
	return nil
}
