package main

import (
	"context"
	"fmt"
	"io"
	"log"

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

	CreateBlog(client)
	// ReadOneBlog(client, id)

	//DeleteBlog(client, "")
	ListBlogs(client)
}

func ListBlogs(client blog.BlogServiceClient) {

	stream, err := client.ListBlogs(context.Background(), &blog.ListBlogsRequest{})
	if err != nil {
		fmt.Println(err)
	}

	for {
		res, err := stream.Recv()
		if err == io.EOF {
			fmt.Println("END OF FILE")
			break
		}
		if err != nil {
			log.Fatalln(err)
		}
		fmt.Printf("Record Found! %+v\n", res.GetBlog())
	}
}

func DeleteBlog(client blog.BlogServiceClient, id string) {
	req := &blog.DeleteBlogRequest{
		BlogId: id,
	}

	res, err := client.DeleteBlog(context.Background(), req)
	if err != nil {
		fmt.Println(err)
	}

	fmt.Println(res)
}

func UpdateBlog(client blog.BlogServiceClient, id string) {

	req := &blog.UpdateBlogRequest{
		Blog: &blog.Blog{
			BlogId:   id,
			AuthorId: 600,
			Title:    "The Wind in the pie",
			Content:  "It is just a farce we eat so much pie..",
		},
	}

	res, err := client.UpdateBlog(context.Background(), req)

	if err != nil {
		log.Fatalln(err)
	}
	fmt.Println(res)
}

func ReadOneBlog(client blog.BlogServiceClient, id string) {

	// reader := bufio.NewReader(os.Stdin)
	// fmt.Print("Enter ID: ")
	// id, _ := reader.ReadString('\n')

	// fmt.Println(id)

	req := &blog.FindBlogRequest{
		BlogId: id,
	}

	res, err := client.FindBlog(context.Background(), req)

	if err != nil {
		log.Fatalln("Something bad happened", err)
	}

	fmt.Println(res)

}

func CreateBlog(client blog.BlogServiceClient) string {
	req := &blog.CreateBlogRequest{
		Blog: &blog.Blog{
			AuthorId: 10,
			Title:    "The Wonder Years",
			Content:  "Sanchez Monreal",
		},
	}

	res, err := client.CreateBlog(context.Background(), req)
	if err != nil {
		log.Fatalln(err)
	}

	fmt.Println("Record Created\n", res)

	return res.Blog.GetBlogId()

}
