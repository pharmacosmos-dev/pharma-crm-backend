#!/bin/bash

read -p "Enter table name >> " table_name
migrate create -ext sql -dir migrations -seq "$table_name"