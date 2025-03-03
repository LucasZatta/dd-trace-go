// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016 Datadog, Inc.

// Code generated by 'go generate ./internal/orchestrion' DO NOT EDIT

//go:build tools

package ddtracego

// Importing "gopkg.in/DataDog/dd-trace-go.v1" in an `orchestrion.tool.go` file
// causes the package to use _all_ available integrations of `dd-trace-go`.
// This makes it easy to ensure all available features of DataDog are enabled in
// your go application, but may cause your dependency closure (`go.mod` and
// `go.sum` files) to include a lot more packages than are stricly necessary for
// your application. If that is a problem, you should instead manually import
// only the specific integrations that are useful to your application.
import (
	_ "gopkg.in/DataDog/dd-trace-go.v1/contrib/99designs/gqlgen"                         // integration
	_ "gopkg.in/DataDog/dd-trace-go.v1/contrib/IBM/sarama.v1"                            // integration
	_ "gopkg.in/DataDog/dd-trace-go.v1/contrib/Shopify/sarama"                           // integration
	_ "gopkg.in/DataDog/dd-trace-go.v1/contrib/aws/aws-sdk-go-v2/aws"                    // integration
	_ "gopkg.in/DataDog/dd-trace-go.v1/contrib/aws/aws-sdk-go/aws"                       // integration
	_ "gopkg.in/DataDog/dd-trace-go.v1/contrib/cloud.google.com/go/pubsub.v1"            // integration
	_ "gopkg.in/DataDog/dd-trace-go.v1/contrib/confluentinc/confluent-kafka-go/kafka"    // integration
	_ "gopkg.in/DataDog/dd-trace-go.v1/contrib/confluentinc/confluent-kafka-go/kafka.v2" // integration
	_ "gopkg.in/DataDog/dd-trace-go.v1/contrib/database/sql"                             // integration
	_ "gopkg.in/DataDog/dd-trace-go.v1/contrib/elastic/go-elasticsearch.v6"              // integration
	_ "gopkg.in/DataDog/dd-trace-go.v1/contrib/gin-gonic/gin"                            // integration
	_ "gopkg.in/DataDog/dd-trace-go.v1/contrib/go-chi/chi"                               // integration
	_ "gopkg.in/DataDog/dd-trace-go.v1/contrib/go-chi/chi.v5"                            // integration
	_ "gopkg.in/DataDog/dd-trace-go.v1/contrib/go-redis/redis"                           // integration
	_ "gopkg.in/DataDog/dd-trace-go.v1/contrib/go-redis/redis.v7"                        // integration
	_ "gopkg.in/DataDog/dd-trace-go.v1/contrib/go-redis/redis.v8"                        // integration
	_ "gopkg.in/DataDog/dd-trace-go.v1/contrib/go.mongodb.org/mongo-driver/mongo"        // integration
	_ "gopkg.in/DataDog/dd-trace-go.v1/contrib/gocql/gocql"                              // integration
	_ "gopkg.in/DataDog/dd-trace-go.v1/contrib/gofiber/fiber.v2"                         // integration
	_ "gopkg.in/DataDog/dd-trace-go.v1/contrib/gomodule/redigo"                          // integration
	_ "gopkg.in/DataDog/dd-trace-go.v1/contrib/google.golang.org/grpc"                   // integration
	_ "gopkg.in/DataDog/dd-trace-go.v1/contrib/gorilla/mux"                              // integration
	_ "gopkg.in/DataDog/dd-trace-go.v1/contrib/gorm.io/gorm.v1"                          // integration
	_ "gopkg.in/DataDog/dd-trace-go.v1/contrib/graph-gophers/graphql-go"                 // integration
	_ "gopkg.in/DataDog/dd-trace-go.v1/contrib/graphql-go/graphql"                       // integration
	_ "gopkg.in/DataDog/dd-trace-go.v1/contrib/hashicorp/vault"                          // integration
	_ "gopkg.in/DataDog/dd-trace-go.v1/contrib/jackc/pgx.v5"                             // integration
	_ "gopkg.in/DataDog/dd-trace-go.v1/contrib/jinzhu/gorm"                              // integration
	_ "gopkg.in/DataDog/dd-trace-go.v1/contrib/julienschmidt/httprouter"                 // integration
	_ "gopkg.in/DataDog/dd-trace-go.v1/contrib/k8s.io/client-go/kubernetes"              // integration
	_ "gopkg.in/DataDog/dd-trace-go.v1/contrib/labstack/echo.v4"                         // integration
	_ "gopkg.in/DataDog/dd-trace-go.v1/contrib/log/slog"                                 // integration
	_ "gopkg.in/DataDog/dd-trace-go.v1/contrib/net/http"                                 // integration
	_ "gopkg.in/DataDog/dd-trace-go.v1/contrib/os"                                       // integration
	_ "gopkg.in/DataDog/dd-trace-go.v1/contrib/redis/go-redis.v9"                        // integration
	_ "gopkg.in/DataDog/dd-trace-go.v1/contrib/segmentio/kafka.go.v0"                    // integration
	_ "gopkg.in/DataDog/dd-trace-go.v1/contrib/sirupsen/logrus"                          // integration
	_ "gopkg.in/DataDog/dd-trace-go.v1/contrib/twitchtv/twirp"                           // integration
	_ "gopkg.in/DataDog/dd-trace-go.v1/contrib/valkey-go"                                // integration
	_ "gopkg.in/DataDog/dd-trace-go.v1/ddtrace/tracer"                                   // integration
	_ "gopkg.in/DataDog/dd-trace-go.v1/internal/civisibility/integrations/gotesting"     // integration
	_ "gopkg.in/DataDog/dd-trace-go.v1/profiler"                                         // integration
)
