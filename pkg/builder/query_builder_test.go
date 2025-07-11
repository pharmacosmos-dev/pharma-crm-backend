package builder

import (
	"reflect"
	"testing"
)

func TestQueryBuilder(t *testing.T) {
	qb := NewQueryBuilder().
		Select("id", "name").
		From("users").
		Where("age > ?", 30).
		OrderBy("name ASC").
		Limit(10)

	sql, args, err := qb.Build()
	expectedSQL := "SELECT id, name FROM users WHERE age > $1 ORDER BY name ASC LIMIT 10"
	expectedArgs := []any{30}

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if sql != expectedSQL {
		t.Errorf("Expected SQL: %s, got: %s", expectedSQL, sql)
	}
	if !reflect.DeepEqual(args, expectedArgs) {
		t.Errorf("Expected args: %v, got: %v", expectedArgs, args)
	}
}

func TestQueryBuilderWithJoinAndCount(t *testing.T) {
	// Test JOIN
	qb1 := NewQueryBuilder().
		Select("u.id", "u.name").
		From("users u").
		Join("INNER JOIN orders o ON u.id = o.user_id").
		Where("u.age > ?", 30).
		OrderBy("u.name ASC").
		Limit(10)

	sql1, args1, err1 := qb1.Build()
	expectedSQL1 := "SELECT u.id, u.name FROM users u INNER JOIN orders o ON u.id = o.user_id WHERE u.age > $1 ORDER BY u.name ASC LIMIT 10"
	expectedArgs1 := []any{30}

	if err1 != nil {
		t.Fatalf("Expected no error, got %v", err1)
	}
	if sql1 != expectedSQL1 {
		t.Errorf("Expected SQL: %s, got: %s", expectedSQL1, sql1)
	}
	if !reflect.DeepEqual(args1, expectedArgs1) {
		t.Errorf("Expected args: %v, got: %v", expectedArgs1, args1)
	}

	// Test COUNT
	qb2 := NewQueryBuilder().
		From("users u").
		Join("LEFT JOIN orders o ON u.id = o.user_id").
		Where("u.age > ?", 30).
		Count()

	sql2, args2, err2 := qb2.Build()
	expectedSQL2 := "SELECT COUNT(*) FROM users u LEFT JOIN orders o ON u.id = o.user_id WHERE u.age > $1"
	expectedArgs2 := []any{30}

	if err2 != nil {
		t.Fatalf("Expected no error, got %v", err2)
	}
	if sql2 != expectedSQL2 {
		t.Errorf("Expected SQL: %s, got: %s", expectedSQL2, sql2)
	}
	if !reflect.DeepEqual(args2, expectedArgs2) {
		t.Errorf("Expected args: %v, got: %v", expectedArgs2, args2)
	}
}
