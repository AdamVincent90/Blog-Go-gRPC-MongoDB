package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"time"

	"../src/blog"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"
)

var collection *mongo.Collection

type server struct{}

type blogItem struct {
	BlogId   primitive.ObjectID `bson:"_id,omitempty"`
	AuthorId int32              `bson:"author_id"`
	Title    string             `bson:"title"`
	Content  string             `bson:"content"`
}

// Server application (THIS CAN BE CONSIDERED A COMPLETLEY SEPERATE APP TO MAIN.GO)
func main() {
	// get code line and number if a fatal error occurs/crashes
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	const (
		protocol = "tcp"             // grpc uses TCP PROTOCOL
		address  = "localhost:50051" // 50051 port represents GRPC PORT
	)

	fmt.Println("Connecting to MongoDB..")

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI("mongodb://localhost:27017"))
	if err != nil {
		log.Fatalln(err)
	}

	fmt.Println("Connected to MongoDB")

	collection = client.Database("blogdb").Collection("blog")

	nl, err := net.Listen(protocol, address) // TAKES IN PROTOCOL AND ADDRESS, LISTENS ON THAT PORT FOR SERVICES
	if err != nil {
		log.Fatalln("Error establishing a", protocol, "connection on", address, "\nerr")
	}

	fmt.Println("GO Server Running at:", address) // if no error then specify to console server is running at address

	s := grpc.NewServer() // creates a pointer to the grpc.server

	blog.RegisterBlogServiceServer(s, &server{}) // registers a server with the grpc server s and the services

	reflection.Register(s)

	go func() {
		if err := s.Serve(nl); err != nil {
			log.Fatalln(err)
		}
	}()
	// Wait for control c for exit
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt)
	// block until a signal is received
	<-ch
	fmt.Println("\nSleeping the server..")
	s.Stop()
	fmt.Println("Server stopped..")
	nl.Close()
	err = client.Disconnect(ctx)
	if err != nil {
		log.Fatalln(err)
	}
	fmt.Println("Mongo Server stopped..")
}

func (*server) ListBlogs(req *blog.ListBlogsRequest, stream blog.BlogService_ListBlogsServer) error {

	mc, err := collection.Find(context.Background(), collection)

	if err != nil {
		return status.Errorf(codes.Internal, fmt.Sprintln("Error finding blogs", err))
	}

	defer mc.Close(context.Background())

	for mc.Next(context.Background()) {
		time.Sleep(1000 * time.Millisecond)
		b := &blogItem{}

		if err := mc.Decode(b); err != nil {
			return status.Errorf(codes.Internal, "Error decoding blogs%+v\n", err)
		}

		if streamErr := stream.Send(&blog.ListBlogResponse{
			Blog: blogItemHelper(b),
		}); streamErr != nil {
			return status.Errorf(codes.Internal, "Error sending blog to client%+v\n", err)
		}
	}

	return nil

}

func (*server) DeleteBlog(ctx context.Context, req *blog.DeleteBlogRequest) (*blog.DeleteBlogResponse, error) {
	id, err := primitive.ObjectIDFromHex(req.GetBlogId())
	if err != nil {
		return nil, status.Errorf(codes.NotFound, fmt.Sprintln("id not found", err))
	}

	filter := bson.M{"_id": id}

	result, deleteErr := collection.DeleteOne(context.Background(), filter)
	if deleteErr != nil {
		return nil, status.Errorf(codes.Internal, fmt.Sprintln("Error deleting record", deleteErr))
	}

	return &blog.DeleteBlogResponse{
		Result: fmt.Sprint(result.DeletedCount, " result deleted from mongoDB database"),
	}, nil
}

func (*server) CreateBlog(ctx context.Context, req *blog.CreateBlogRequest) (*blog.CreateBlogResponse, error) {
	d := req.GetBlog()
	data := blogItem{
		AuthorId: d.GetAuthorId(),
		Title:    d.GetTitle(),
		Content:  d.GetContent(),
	}

	result, err := collection.InsertOne(context.Background(), data)
	if err != nil {
		log.Fatalln(err)
	}

	oid, ok := result.InsertedID.(primitive.ObjectID)
	if !ok {
		log.Fatalln(ok)
	}

	res := &blog.CreateBlogResponse{
		Blog: &blog.Blog{
			BlogId:   oid.Hex(),
			AuthorId: d.GetAuthorId(),
			Title:    d.GetTitle(),
			Content:  d.GetContent(),
		},
	}

	fmt.Println("Blog created and stored to mongodb database")

	return res, nil
}

func (*server) UpdateBlog(ctx context.Context, req *blog.UpdateBlogRequest) (*blog.UpdateBlogResponse, error) {

	b := req.GetBlog()

	id, err := primitive.ObjectIDFromHex(b.GetBlogId())

	if err != nil {
		log.Fatalln(err)
	}

	filter := bson.M{"_id": id}
	result := collection.FindOne(context.Background(), filter)

	bi := &blogItem{}

	if err2 := result.Decode(&bi); err2 != nil {
		log.Fatalln(err2)
	}

	bi.AuthorId = b.GetAuthorId()
	bi.Title = b.GetTitle()
	bi.Content = b.GetContent()

	_, err3 := collection.ReplaceOne(context.Background(), filter, bi)

	if err3 != nil {
		fmt.Println(err3)
	}

	return &blog.UpdateBlogResponse{
		Blog: blogItemHelper(bi),
	}, nil
}

func blogItemHelper(b *blogItem) *blog.Blog {
	return &blog.Blog{
		BlogId:   b.BlogId.Hex(),
		AuthorId: b.AuthorId,
		Title:    b.Title,
		Content:  b.Content,
	}
}

func (*server) FindBlog(ctx context.Context, req *blog.FindBlogRequest) (*blog.FindBlogResponse, error) {
	fmt.Println("Finding Blog service invoked...")
	id := req.GetBlogId() // GET BLOG ID FROM REQUEST

	if id == "" {
		fmt.Println("The ID entered is empty, returning error to client")
		return nil, status.Errorf(codes.NotFound, "The ID entered is empty, please submit a ID..")
	}

	oid, err := primitive.ObjectIDFromHex(id) // CONVERT BLOG ID FROM STRING INTO BSON
	if err != nil {
		fmt.Println("ID entered, invalid, returning error to client")
		return nil, status.Errorf(codes.InvalidArgument, "The ID entered is invalid, please submit a valid ID")
	}

	b := &blogItem{}
	filter := bson.M{"_id": oid}

	fmt.Println("Finding record with objectID....")
	res := collection.FindOne(ctx, filter)

	if err := res.Decode(&b); err != nil {
		fmt.Println("Error occured during decode, sending to client.")
		return nil, status.Errorf(codes.NotFound, "No Record with ID was matched...")
	}

	fmt.Println("Found record.. sending to client")

	return &blog.FindBlogResponse{
		Blog: blogItemHelper(b),
	}, nil

}
