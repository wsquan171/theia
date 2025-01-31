// Copyright 2022 Antrea Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"database/sql"
	"fmt"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
)

func TestMonitor(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	t.Run("testMonitorMemoryWithDeletion", func(t *testing.T) {
		testMonitorMemoryWithDeletion(t, db, mock)
	})
	t.Run("testMonitorMemoryWithoutDeletion", func(t *testing.T) {
		testMonitorMemoryWithoutDeletion(t, db, mock)
	})
	t.Run("testGetDeleteRowNum", func(t *testing.T) {
		testGetDeleteRowNum(t, db, mock)
	})
}

func testMonitorMemoryWithDeletion(t *testing.T, db *sql.DB, mock sqlmock.Sqlmock) {
	baseTime := time.Now()
	diskRow := sqlmock.NewRows([]string{"free_space", "total_space"}).AddRow(4, 10)
	countRow := sqlmock.NewRows([]string{"count"}).AddRow(10)
	timeRow := sqlmock.NewRows([]string{"timeInserted"}).AddRow(baseTime.Add(5 * time.Second))
	mock.ExpectQuery("SELECT free_space, total_space FROM system.disks").WillReturnRows(diskRow)
	mock.ExpectQuery("SELECT COUNT() FROM flows").WillReturnRows(countRow)
	mock.ExpectQuery("SELECT timeInserted FROM flows LIMIT 1 OFFSET 5").WillReturnRows(timeRow)
	for _, table := range []string{"flows", "flows_pod_view", "flows_node_view", "flows_policy_view"} {
		command := fmt.Sprintf("ALTER TABLE %s DELETE WHERE timeInserted < toDateTime('%v')", table, baseTime.Add(5*time.Second).Format(timeFormat))
		mock.ExpectExec(command).WillReturnResult(sqlmock.NewResult(0, 5))
	}

	tableName = "flows"
	mvNames = []string{"flows_pod_view", "flows_node_view", "flows_policy_view"}
	monitorMemory(db)

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}

}

func testMonitorMemoryWithoutDeletion(t *testing.T, db *sql.DB, mock sqlmock.Sqlmock) {
	diskRow := sqlmock.NewRows([]string{"free_space", "total_space"}).AddRow(6, 10)
	mock.ExpectQuery("SELECT free_space, total_space FROM system.disks").WillReturnRows(diskRow)

	monitorMemory(db)

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}

func testGetDeleteRowNum(t *testing.T, db *sql.DB, mock sqlmock.Sqlmock) {
	countRow := sqlmock.NewRows([]string{"count"}).AddRow(10)
	mock.ExpectQuery("SELECT COUNT() FROM flows").WillReturnRows(countRow)

	tableName = "flows"
	deleteRowNumber, err := getDeleteRowNum(db)

	assert.Equalf(t, uint64(5), deleteRowNumber, "Got deleteRowNumber %d, expect %d", deleteRowNumber, 5)
	assert.NoErrorf(t, err, "getDeleteRowNum returns error %v", err)

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}
