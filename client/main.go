package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"

	"../src/blog"
	"google.golang.org/grpc"
)

func main() {
	fmt.Println("Running client...")

	cc, err := grpc.Dial("localhost:50051", grpc.WithInsecure())
	if err != nil {
		log.Fatalln(err)
	}
	defer cc.Close()

	client := blog.NewBlogServiceClient(cc)

	ReadOneBlog(client)
}

func ReadOneBlog(client blog.BlogServiceClient) {

	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Enter ID: ")
	id, _ := reader.ReadString('\n')

	fmt.Println(id)

	req := &blog.FindBlogRequest{
		BlogId: id,
	}

	res, err := client.FindBlog(context.Background(), req)

	if err != nil {
		log.Fatalln("Something bad happened", err)
	}

	fmt.Println(res)

}

func CreateBlog(client blog.BlogServiceClient) {
	req := &blog.CreateBlogRequest{
		Blog: &blog.Blog{
			AuthorId: 4,
			Title:    "Second",
			Content:  "Third",
		},
	}

	res, err := client.CreateBlog(context.Background(), req)
	if err != nil {
		log.Fatalln(err)
	}

	fmt.Println("Record Created\n", res)

}
