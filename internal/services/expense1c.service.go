package services

import "github.com/pharma-crm-backend/domain"

func (s *Services) CreateNewExpense(cashboxData *domain.OperationWithStore, docsNumber string) error {
	query := `INSERT INTO shift_expenses(store_id, cashbox_id, cashbox_operation_id, docs_number) VALUES(?, ?, ?, ?)`
	err := s.db.Exec(query, cashboxData.StoreId, cashboxData.CashboxId, cashboxData.Id, docsNumber).Error
	if err != nil {
		s.log.Warn("ERROR on creating shift_expenses: %v", err)
		return err
	}

	return nil
}

func (s *Services) UpdateExpenseStatusByDocNumber(status int, docsNumber string) error {
	query := `UPDATE shift_expenses SET status = ? WHERE docs_number = ?`
	err := s.db.Exec(query, status, docsNumber).Error
	if err != nil {
		s.log.Warn("ERROR on updating shift_expenses status: %v", err)
		return err
	}
	return nil
}
