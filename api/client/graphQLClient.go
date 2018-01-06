package client

import (
  "os"
  "fmt"
  "log"
  "bytes"
  "net/http"
  "encoding/json"
  "github.com/olekukonko/tablewriter"
)

type GQLQuery struct {
  Query string `json:"query"`
  Variables interface{} `json:"variables"`
}

func GetGraphQLApiURL(server string) (string) {
  if server == "github.com" {
    return fmt.Sprintf("https://api.%s/graphql", server)
  } else {
    return fmt.Sprintf("https://%s/api/graphql", server)
  }
}

func GraphQLQueryAndPrintTable(server, token, query string, params map[string]interface{}, responseHandler GraphQLResponseHandler) {
  table := tablewriter.NewWriter(os.Stdout)
  table.SetHeader(responseHandler.TableHeader())
  paginatedGraphQLQueryAndPrintTable(server, token, query, params, table, responseHandler)
  table.Render()
}

func paginatedGraphQLQueryAndPrintTable(server, token, query string, params map[string]interface{}, table *tablewriter.Table, responseHandler GraphQLResponseHandler) {
  if params["count"] == nil {
    params["count"] = 100
  }
  graphQLQuery := GQLQuery{query, params}
  jsonValue, _ := json.Marshal(graphQLQuery)
  apiURL := GetGraphQLApiURL(server)
  req, err := http.NewRequest("POST", apiURL, bytes.NewBuffer(jsonValue))
  if err != nil {
    log.Fatal("Failed while building the HTTP client: ", err)
    return
  }

  // Provide authentication
  req.Header.Add("Authorization", fmt.Sprintf("bearer %s", token))

  client := http.Client{}
  resp, err := client.Do(req)
  if err != nil {
    log.Fatal("Error while querying the server.", err)
    return
  } else if resp.StatusCode != http.StatusOK {
    log.Fatalf("Ooops... sorry, server sent a %d HTTP status code: %s", resp.StatusCode, http.StatusText(resp.StatusCode))
    return
  }

  // Close when method returns
  defer resp.Body.Close()

  var jsonObj map[string]interface{}

  // Decode the JSON array
  decodeError := json.NewDecoder(resp.Body).Decode(&jsonObj)
  if decodeError != nil {
    fmt.Printf("Error while decoding the server response: %s", decodeError)
    return
  } else {
    table.AppendBulk(responseHandler.TableRows(jsonObj))
    hasNextPage, endCursor := getPageInfo(jsonObj, responseHandler.ResultPath())
    if hasNextPage {
      params["cursor"] = endCursor
      paginatedGraphQLQueryAndPrintTable(server, token, query, params, table, responseHandler)
    }
  }
}

// Navigate the JSON response to retrive the 'pageInfo' object and return its prorperies (hasNextPage and endCursor)
func getPageInfo(jsonObj map[string]interface{}, path []string) (hasNextPage bool, endCursor string) {
  if(len(path) == 0) {
    pageInfo := jsonObj["pageInfo"].(map[string]interface{})
    if pageInfo["hasNextPage"].(bool) {
      return true, pageInfo["endCursor"].(string)
    } else {
      return false, ""
    }
  } else {
    return getPageInfo(jsonObj[path[0]].(map[string]interface{}), path[1:])
  }
}


func GraphQLQuery(server, token, query string, params map[string]interface{}) (resp *http.Response, err error) {
  graphQLQuery := GQLQuery{query, params}
  jsonValue, _ := json.Marshal(graphQLQuery)
  apiURL := GetGraphQLApiURL(server)
  req, err := http.NewRequest("POST", apiURL, bytes.NewBuffer(jsonValue))
  if err != nil {
    log.Fatal("Failed while building the HTTP client: ", err)
    return
  }

  // Provide authentication
  req.Header.Add("Authorization", fmt.Sprintf("bearer %s", token))

  client := http.Client{}
  return client.Do(req)
}


func GraphQLQueryObject(server, token, query string, params map[string]interface{}) map[string]interface{} {
  resp, err := GraphQLQuery(server, token, query, params)
  if err != nil {
    log.Fatal("Error while querying the server.\n", err)
    return nil
  } else if resp.StatusCode != http.StatusOK {
    log.Fatalf("Ooops... sorry, server sent a %d HTTP status code: %s\n", resp.StatusCode, http.StatusText(resp.StatusCode))
    return nil
  }

  // Close when method returns
  defer resp.Body.Close()

  var jsonObj map[string]interface{}

  // Decode the JSON array
  decodeError := json.NewDecoder(resp.Body).Decode(&jsonObj)
  if decodeError != nil {
    fmt.Printf("Error while decoding the server response: %s", decodeError)
    return nil
  } else {
    return jsonObj
  }
}
