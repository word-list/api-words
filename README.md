# api-words

## Description

This repo contains the source code for the `api-words` lambda, which is part of the Word List application's API provision.  This is an HTTP-triggered lambda which responds to requests from API Gateway.

## Environment Variables

The lambda uses the following environment variables:

| Variable Name        | Description                                              |
|----------------------|----------------------------------------------------------|
| DB_CONNECTION_STRING | Connection string for the database containing the words. |
