package main

import (
	"context"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/tealeg/xlsx/v3"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
)

type App struct {
	Router *gin.Engine
	DB     *pgxpool.Pool
}

type Request struct {
	ID  int    `form:"id"`
	URL string `form:"url"`
}

type StructTest struct {
	ID      int    `xlsx:"0"`
	Name    string `xlsx:"1"`
	Price   int    `xlsx:"2"`
	Amount  int    `xlsx:"3"`
	BoolVal bool   `xlsx:"4"`
}

type Status struct {
	Inserts int
	Updates int
	Deletes int
	Errors  int
}

func (app *App) Initialize() {
	fmt.Println(os.Getenv("POSTGRES_URL"))
	var err error
	app.DB, err = pgxpool.Connect(context.Background(), os.Getenv("POSTGRES_URL"))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to connect to database: %v\n", err)
		os.Exit(1)
	}

	query := `CREATE TABLE IF NOT EXISTS PRODUCTS (
		name TEXT NOT NULL,
		offerid INT NOT NULL,
		price int NOT NULL,
		quantity INT check (quantity >= 0) NOT NULL,
		seller INT check (seller >= 0) NOT NULL
		);
	`
	_, err = app.DB.Exec(context.Background(), query)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to create table: %v\n", err)
		os.Exit(1)
	}

	query = `CREATE UNIQUE INDEX IF NOT EXISTS idx_products
			 ON products(OfferID, Seller);`

	_, err = app.DB.Exec(context.Background(), query)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to create UNIQUE INDEX: %v\n", err)
		os.Exit(1)
	}

	app.Router = gin.Default()

	app.initializeRoutes()
}

func (app *App) Run() {
	app.Router.Run(":8080")
}

func (app *App) initializeRoutes() {

	app.Router.GET("/", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"data": "hello world"})
	})

	app.Router.GET("/products", app.getProducts)
	app.Router.POST("/updateProduct", app.postProducts)
	app.Router.GET("/status", app.getStatus)

}

func (app *App) getStatus(c *gin.Context) {
	c.String(200, "Success")
}
func (app *App) getProducts(c *gin.Context) {
	seller := c.DefaultQuery("seller_id", "")
	offerId := c.DefaultQuery("offer_id", "")
	name := c.DefaultQuery("name", "")
	fmt.Println(seller, offerId, name)
	offerIdUint, _ := strconv.ParseUint(offerId, 10, 64)
	sellerUint, _ := strconv.ParseUint(seller, 10, 64)
	p := product{Name: name, OfferID: int(offerIdUint), Seller: int(sellerUint)}
	_, err := p.getProducts(app.DB)
	if err != nil {
		switch err {
		case pgx.ErrNoRows:
			c.String(200, "Products not found")
		default:
			c.String(500, "Server error")
		}
		return
	}
	c.String(200, "Success")
}

func (app *App) postProducts(c *gin.Context) {
	var req Request
	if c.ShouldBind(&req) == nil {
		log.Println(req.ID)
		log.Println(req.URL)
	}

	out, err := os.Create("file.xlsx")
	if err != nil {
		fmt.Println("Error creating file.xlsx")
		return
	}
	defer out.Close()

	resp, err := http.Get(req.URL)
	if err != nil {
		fmt.Println("Error getting URL")
		return
	}
	defer resp.Body.Close()

	_, err = io.Copy(out, resp.Body)

	xlFile, err := xlsx.OpenFile("file.xlsx")
	if err != nil && xlFile != nil {
		fmt.Println("Error reading file.xlsx")
	}
	status := Status{0, 0, 0, 0}
	for _, sheet := range xlFile.Sheets {
		err = sheet.ForEachRow(func(r *xlsx.Row) error {
			readStruct := &StructTest{}
			err := r.ReadStruct(readStruct)
			if err != nil {
				panic(err)
			}
			p := product{Name: readStruct.Name, Price: readStruct.Price, Quantity: readStruct.Amount, Seller: req.ID, OfferID: readStruct.ID}
			if readStruct.BoolVal {
				if p.rowExists(app.DB) {
					if err = p.updateProduct(app.DB); err != nil {
						fmt.Println("Error in updateProduct: ", err.Error())
						status.Errors++
						return err
					} else {
						status.Updates++
					}
				} else if err = p.createProduct(app.DB); err != nil {
					fmt.Println("Error in createProduct: ", err.Error())
					status.Errors++
					return err
				} else {
					status.Inserts++
				}
			} else {
				if err = p.deleteProduct(app.DB); err != nil {
					fmt.Println("Error in deleteProduct: ", err.Error())
					status.Errors++
					return err
				} else {
					status.Deletes++
				}
			}
			return nil
		})
		if err != nil {
			fmt.Println("Error in sheet.ForEachRow(app.rowVisitor): ", err)
		}
		c.JSON(http.StatusOK, gin.H{"Inserts": status.Inserts, "Updates": status.Updates, "Deletes": status.Deletes, "Errors": status.Errors})
	}
	err = os.Remove("file.xlsx")
	if err != nil {
		fmt.Println("Error removing file.xlsx")
	}
}
