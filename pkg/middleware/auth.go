package middleware

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/golang-jwt/jwt/v4"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// ContextKey,  is a type for context values.
type ContextKey string

// UserIDKey, is used for storing user ID in context.
const UserIDKey ContextKey = "userID"

// AuthInterceptor, authenticates incoming grpc requests.
func AuthInterceptor(jwtSecret string) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler) (interface{}, error) {
		// skip authentication for login and register for they are public
		if info.FullMethod == "/proto.UserService/Login" || info.FullMethod == "proto.UserService/Register" {
			return handler(ctx, req)
		}

		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			log.Printf("[Server Interceptor: %s] ERROR: Could not found metadata from incoming context",
				info.FullMethod)

			return nil, status.Errorf(codes.Unauthenticated, "could not found metadata")
		}

		log.Printf("[Server Interceptor: %s], metadata received: %v", info.FullMethod, md)

		authHeaders := md.Get("authorization")
		if len(authHeaders) == 0 {
			log.Printf("[Server Interceptor: %s]ERROR: could not found authorization header",
				info.FullMethod)

			return nil, status.Errorf(codes.Unauthenticated, "could not found authorization header")
		}

		tokenString := strings.TrimPrefix(authHeaders[0], "Bearer ")
		if tokenString == "" {
			return nil, status.Errorf(codes.Unauthenticated, "could not found authorization token")
		}

		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}

			return []byte(jwtSecret), nil
		})

		if err != nil {
			return nil, status.Errorf(codes.Unauthenticated, "invalid token: %v", err)
		}

		if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
			userID, ok := claims["user_id"].(string)
			if !ok {
				return nil, status.Errorf(codes.Unauthenticated, "invalid user_id in token")
			}

			newCtx := context.WithValue(ctx, UserIDKey, userID)

			return handler(newCtx, req)
		}

		return nil, status.Error(codes.Unauthenticated, "invalid token")
	}
}
