package main

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
)

type product struct {
	Name     string `json:"name"`
	Price    int    `json:"price"`
	Quantity int    `json:"quantity"`
	Seller   int    `json:"seller"`
	OfferID  int    `json:"id in sellers system"`
}

func itos(b int) string {
	if b != 0 {
		return "1"
	}
	return "0"
}

func stos(b string) string {
	if len(b) > 0 {
		return "1"
	}
	return "0"
}

func (p *product) getProducts(db *pgxpool.Pool) ([]product, error) {
	var code string
	var rows pgx.Rows
	var err error
	code = itos(p.Seller) + itos(p.OfferID) + stos(p.Name)
	switch code {
	case "111":
		fmt.Println("111")
		rows, err = db.Query(context.Background(), "SELECT * FROM products WHERE Seller=$1 AND offerid=$2 AND name LIKE $3", p.Seller, p.OfferID, p.Name+"%")
	case "011":
		fmt.Println("011")
		rows, err = db.Query(context.Background(), "SELECT * FROM products WHERE offerid=$1 AND name LIKE '$2%'", p.OfferID, p.Name+"%")
	case "001":
		fmt.Println("001")
		rows, err = db.Query(context.Background(), "SELECT * FROM products WHERE name LIKE $1", p.Name+"%")
	case "010":
		fmt.Println("010")
		rows, err = db.Query(context.Background(), "SELECT * FROM products WHERE offerid=$1", p.OfferID)
	case "100":
		fmt.Println("100")
		rows, err = db.Query(context.Background(), "SELECT * FROM products WHERE Seller=$1", p.Seller)
	case "101":
		fmt.Println("101")
		rows, err = db.Query(context.Background(), "SELECT * FROM products WHERE Seller=$1 AND name LIKE '$2%'", p.Seller, p.Name+"%")
	case "110":
		fmt.Println("110")
		rows, err = db.Query(context.Background(), "SELECT * FROM products WHERE Seller=$1 AND offerid=$2", p.Seller, p.OfferID)
	default:
		fmt.Println("DEFAULT")
		rows, err = db.Query(context.Background(), "SELECT * FROM products")
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	products := []product{}

	for rows.Next() {
		var p product
		if err := rows.Scan(&p.Name, &p.Price, &p.Quantity, &p.Seller, &p.OfferID); err != nil {
			return nil, err
		}
		products = append(products, p)
	}
	return products, nil
}

func (p *product) updateProduct(db *pgxpool.Pool) error {
	_, err :=
		db.Exec(context.Background(), "UPDATE products SET Price=$1, Quantity=$2 WHERE Seller=$3 AND offerid=$4",
			p.Price, p.Quantity, p.Seller, p.OfferID)
	return err
}

func (p *product) deleteProduct(db *pgxpool.Pool) error {
	_, err := db.Exec(context.Background(), "DELETE FROM products WHERE Seller=$1 AND offerid=$2", p.Seller, p.OfferID)
	return err
}

func (p *product) createProduct(db *pgxpool.Pool) error {
	fmt.Println(string(p.Name), p.OfferID, p.Price, p.Quantity, p.Seller)
	_, err := db.Exec(context.Background(), "INSERT INTO products(Name, OfferID, Price, Quantity, Seller) VALUES($1, $2, $3, $4, $5)",
		&p.Name, &p.OfferID, &p.Price, &p.Quantity, &p.Seller)
	if err != nil {
		return err
	}
	return nil
}

func (p *product) rowExists(db *pgxpool.Pool) bool {
	var exists bool
	err := db.QueryRow(context.Background(), "SELECT exists (SELECT Name FROM products WHERE Seller=$1 AND offerid=$2)", p.Seller, p.OfferID).Scan(&exists)
	if err != nil && err != sql.ErrNoRows {
		fmt.Println("error checking if row exists %v", err)
	}
	return exists
}
