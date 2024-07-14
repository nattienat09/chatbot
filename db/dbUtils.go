package dbUtils

import (
        "fmt"
        "log"
        "database/sql"
        "github.com/go-sql-driver/mysql"
)


var db *sql.DB


func InitDb(dbuser string, dbpass string) {//*sql.DB {
    cfg := mysql.Config{
        User:   dbuser,
        Passwd: dbpass,
        Net:    "tcp",
        Addr:   "127.0.0.1:3306",
        DBName: "chatbot",
    }
    // Get a database handle.
    var sql_err error
    db, sql_err = sql.Open("mysql", cfg.FormatDSN())
    if sql_err != nil {
        log.Fatal(sql_err)
    }

    pingErr := db.Ping()
    if pingErr != nil {
        log.Fatal(pingErr)
    }
    fmt.Println("Connected to DB!")
    //return db
}

func GetProductNameFromProductId(productId int) (string,error){
	var productName string
	query := "SELECT product_name FROM products WHERE product_id = ?"
	err := db.QueryRow(query, productId).Scan(&productName)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", fmt.Errorf("No product found with id %d", productId)
		}
		return "", fmt.Errorf("Failed to query product name: %v", err)
	}
	return productName, nil
}


func SaveCustomerRating(customerId int, productId int, rating int) error {
	fmt.Printf("Inserted rating.")
	query := "INSERT INTO reviews (customer_id, product_id, rating) VALUES (?,?,?)"
	stmt, err := db.Prepare(query)
	if err != nil {
		return fmt.Errorf("Failed to prepare statement: %v",err)
	}
	defer stmt.Close()
	_, err = stmt.Exec(customerId, productId, rating)
	if err != nil {
                return fmt.Errorf("Failed to insert in database: %v",err)
        }
	fmt.Printf("Inserted rating.")
	return nil
}
