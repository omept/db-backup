package main

import (
	"archive/zip"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"sync"

	_ "github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
	// "github.com/subosito/gotenv"
)

// func init() {
// 	gotenv.Load()
// }

var (
	dbUsername = "root"
	dbPassword = "password"
	dbHost     = "127.0.0.1"
	dbPort     = "3306"
	dbName     = "yapdoof"
)

func main() {

	// List of tables to be ignored during backup
	ignoredTables := []string{"api_logs"}

	// Output path for backups
	outputPath := "./backups/"
	sourceDir := "./backups/"
	zipFilePath := "./archive.zip"

	// Database connection string
	// username:password@protocol(address)/dbname?param=value
	dbConnString := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s", dbUsername, dbPassword, dbHost, dbPort, dbName)
	// dbConnString := "username:password@tcp(localhost:3306)/dbname?charset=utf8mb4&parseTime=True&loc=Local"

	// Open a connection to the database
	db, err := sqlx.Open("mysql", dbConnString)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Get the list of tables in the database
	tables, err := getTables(db)
	if err != nil {
		log.Fatal(err)
	}

	// Create a wait group to wait for all goroutines to finish
	var wg sync.WaitGroup
	var wg2 sync.WaitGroup

	// Create a channel to signal the worker pool to stop
	stopChan := make(chan bool, 2)

	// Create a channel to send table names to workers
	tableChan := make(chan string, len(tables))

	// Start worker pool
	for i := 0; i < 2; i++ { // Number of worker goroutines
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case tableName := <-tableChan:
					// Backup each table if it is not in the ignoredTables list

					if !contains(ignoredTables, tableName) && len(tableName) > 0 {
						err := backupTable(db, tableName, outputPath)
						if err != nil {
							log.Printf("Error backing up table %s: %s\n", tableName, err)
						} else {
							log.Printf("Table %s backed up successfully\n", tableName)
						}
					}
					wg2.Done()

				case <-stopChan:
					return
				}
			}
		}()
	}

	// Send table names to the worker pool
	for _, table := range tables {
		wg2.Add(1)
		tableChan <- table
	}

	// Wait for all worker goroutines to finish
	wg2.Wait()

	stopChan <- true
	stopChan <- true

	// Wait for all worker goroutines to finish
	wg.Wait()

	// Create a Zip archive
	if err := ZipDir(sourceDir, zipFilePath); err != nil {
		fmt.Printf("Error zipping directory: %s\n", err)
		return
	}

	log.Println("Backup completed and compressed successfully")
}

// backupTable backs up a given table to a file using mysqldump
func backupTable(db *sqlx.DB, tableName, outputPath string) error {
	// Create a file for each table backup
	filePath := fmt.Sprintf("%s%s_backup.sql", outputPath, tableName)
	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	// Use mysqldump to dump the entire table content to the file
	cmd := exec.Command("mysqldump", "--skip-comments", "-u", dbUsername, "-p"+dbPassword, dbName, tableName)
	cmd.Stdout = file

	if err := cmd.Run(); err != nil {
		return err
	}

	log.Printf("Table %s backed up to %s\n", tableName, filePath)
	return nil
}

// getTables retrieves the list of tables in the database
func getTables(db *sqlx.DB) ([]string, error) {
	var tables []string
	err := db.Select(&tables, "SHOW TABLES")
	return tables, err
}

// ZipDir creates archive
func ZipDir(sourceDir, zipFilePath string) error {
	zipFile, err := os.Create(zipFilePath)
	if err != nil {
		return err
	}
	defer zipFile.Close()

	zipWriter := zip.NewWriter(zipFile)
	defer zipWriter.Close()

	err = filepath.Walk(sourceDir, func(filePath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		relPath, err := filepath.Rel(sourceDir, filePath)
		if err != nil {
			return err
		}

		// Create a new file header
		header, err := zip.FileInfoHeader(info)
		if err != nil {
			return err
		}

		// Set the header name to the relative path within the zip file
		header.Name = relPath

		// Create a new file in the zip archive
		zipFile, err := zipWriter.CreateHeader(header)
		if err != nil {
			return err
		}

		// Open the source file
		file, err := os.Open(filePath)
		if err != nil {
			return err
		}
		defer file.Close()

		// Copy the file contents to the zip file
		_, err = io.Copy(zipFile, file)
		return err
	})

	return err
}

// contains checks if a string is present in a slice of strings
func contains(slice []string, str string) bool {
	for _, s := range slice {
		if s == str {
			return true
		}
	}
	return false
}
