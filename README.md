# gomux
Go mux is a wrapper for gorilla mux that allows for quicker spin ups of configured https servers with cors handling.

## Requirements

You will need a tls.crt and a tls.key for the server to run. By default this package looks for `server.crt` and `server.key` wherever the binary is running from.

## Examples

```go
package main

func main() {
	mux := gomux.New(context.Background(), "api", gomux.TLS(), gomux.Port(10000))

	mux.AddRoutes(
		gomux.Get("/users", Users),
		gomux.Post("/user", SaveUser),
		gomux.Delete("/user/{userid}", DeleteUser),
	)

	log.Fatal(mux.Serve())
}
```

With just those few lines you have a TLS server running with basic cors handling implemented. 

**WARNING**

**Review the default settings in this code to be sure that it is appropriate for the environment and situation you intend to run your code. This code may or may not be production ready and is subject to change during any particular release!**

---

Lets have a look at the Users function signature

```go
type User struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

func Users(w io.Writer, r *http.Request) (interface{}, error) {
	return &User{ID: 1, Name: "Hunter"}, nil
}
```

This snippet of code will return a `200` with this body when hitting
`GET https://localhost:10000/api/users`

```json
{
  "id": 1,
  "name": "Hunter"
}
```

The marshaling and formatting of the response body and the response code was all handled by the gomux package for you! 

---

Now, let's look at the SaveUser function and how it handles it's error state.

```go
func SaveUser(w io.Writer, r *http.Request) (interface{}, error) {
	fmt.Println("User saving will error as an example")
	return nil, errors.E(errors.CodeBadRequest, errors.Invalid, "Invalid User ID")
}
```
*refer to github.com/hunterdishner/errors for the specifics of the error package this module relies on*

Making a call to that route will return you a `400` response code with this body

```json
{
  "code": 400,
  "op": "",
  "kind": 1,
  "err": "Invalid User ID",
  "stack": [
    "test/main.go:37 return nil, errors.E(errors.CodeBadRequest, errors.Invalid, \"Invalid User ID\")",
    "github.com/hunterdishner/gomux@v0.0.0-20201007225513-ee88fc69884c/gomux.go:231 data, err := fn(w, r)",
    "net/http/server.go:2012 f(w, r)"
  ]
}
```

### Note
The response code that gomux decides to return is based off of what code it finds in the error you're returning. If the error is not of type errors.Error then it will wrap the error automatically and use a `500` response code instead. 

---


Last but not least we can look at `DeleteUser`

```go
func DeleteUser(w io.Writer, r *http.Request) (interface{}, error) {
	//assume delete succeedes but we want to return nothing but a 200
	return nil, nil
}
```

This code will simply return a `200` response with no body at all.


---

## What if you need to go back to the standard way of using Gorilla Mux?

This can be accomplished by adding the line
```go
gomux.GetFn("/example", Example),
```

and creating a function with the original signature of 

```go
func Example(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(202)
	w.Write([]byte("Successful Call!"))
}
```

---

There you go! If you have any improvements or suggestions please open an issue and I'll address them as they come. The project is still a work in progress but I am deeming it "production ready" with the caveat that the default tls and cors configurations will most likely not work for everyone.


