package services

import "gorm.io/gorm"

func (s *Storage) CreateEmployeeBonus(tx *gorm.DB, employeeId string, saleId string, cashBoxOperationID string) error {
	err := tx.Exec(`
    INSERT INTO
        employee_bonus (employee_id, sale_id, cashbox_operation_id, bonus_amount)
    VALUES (?, ?, ?,
            (select sum(sp.bonus_amount) 
            FROM cart_items ci 
            JOIN store_products sp 
            ON sp.id = ci.store_product_id 
            WHERE ci.sale_id=?))`, employeeId, saleId, cashBoxOperationID, saleId).Error
	if err != nil {
		s.log.Error(err)
		return err
	}
	return nil
}
