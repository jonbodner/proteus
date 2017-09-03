package proteus

import "testing"

func TestEmbeddedNoSql(t *testing.T) {
	type InnerEmbeddedProductDao struct {
		Insert func(e Executor, p Product) (int64, error) `proq:"insert into Product(name) values(:p.Name:)" prop:"p"`
	}

	type OuterEmbeddedProductDao struct {
		InnerEmbeddedProductDao
		FindById func(e Querier, id int64) (Product, error) `proq:"select * from Product where id = :id:" prop:"id"`
	}

	productDao := OuterEmbeddedProductDao{}
	err := Build(&productDao, Sqlite)
	if err != nil {
		t.Fatal(err)
	}
	if productDao.FindById == nil {
		t.Fatal("should have populated findById")
	}
	if productDao.Insert == nil {
		t.Fatal("should have populated insert")
	}
}
