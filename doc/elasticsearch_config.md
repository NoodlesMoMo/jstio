
## Query Kibana index-patten id

    GET .kibana/_search
    {
      "from": 0,
      "size": 50,
      "_source": "index-pattern.title",
      "query": {
        "term": {
          "type": "index-pattern"
        }
      }
    }
    
    
