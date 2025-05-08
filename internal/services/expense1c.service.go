package services

func (s *Services) CreateNewExpense(storeID string, docsNumber string) error {
	query := `INSERT INTO shift_expenses(store_id, docs_number) VALUES(?, ?)`
	err := s.db.Exec(query, storeID, docsNumber).Error
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
