package storage

func (s *Storage) CheckFieldEmployee(field, value string) (bool, error) {
	var temp = 0
	err := s.db.Raw(`SELECT 1 FROM employees WHERE `+field+` = ?`, value).Scan(&temp).Error
	if err != nil {
		return false, err
	}
	return false, nil
}
