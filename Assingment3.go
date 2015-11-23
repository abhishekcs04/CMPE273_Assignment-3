package main

import  ("github.com/julienschmidt/httprouter"
		 "io/ioutil"
		 "net/http"
		 "encoding/json"
		 "fmt"
		 "gopkg.in/mgo.v2"
         "gopkg.in/mgo.v2/bson"
         "log"
         "strconv"
         "sort"
         "bytes")

func main() {
	mux := httprouter.New()
    mux.POST("/trips",PostTriplocation)
    mux.GET("/trips/:tripid",GetTripulocation)
    mux.PUT("/trips/:tripid/request",PutTriplocation)
    server := http.Server{
            Addr:        "0.0.0.0:8080",
            Handler: mux,
    }
    server.ListenAndServe()
}


func (a ByTotal) Len() int           { return len(a) }
func (a ByTotal) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ByTotal) Less(i, j int) bool { return a[i].Total < a[j].Total }

const key = ""

type Request struct{
	Starting_from_location_id string `json:"starting_from_location_id"`
	Location_ids [] string `json:"location_ids"` 
}

type Message struct{
	Msg string `json:"Message"`
}

type Database struct{
	Coor Coordinate `json:"coordinate" bson:"coordinate"`
}

type Coordinate struct{
	Lat float64 `json:"lat" bson:"lat"`
	Lng float64 `json:"lng" bson:"lng"`
}



type OrderedSlice struct{
	Location_id string
	Cost float64
	Distance float64
	Duration float64
	Total float64
	Req_id string
}

type DataId struct{
	Id int  `bson:"id"`
}

type ByTotal []OrderedSlice



type Response struct{
	Id int `json:"id" bson:"id"`
	Status string `json:"status" bson:"status"`
	Starting_from_location_id string `json:"starting_from_location_id" bson:"starting_from_location_id"`
	Next_destination_location_id string `json:"next_destination_location_id,omitempty"`
	Best_route_location_ids [] string `json:"best_route_location_ids" bson:"best_route_location_ids"`
	Total_uber_costs float64 `json:"total_uber_costs"  bson:"total_uber_costs"`
	Total_uber_duration float64 `json:"total_uber_duration" bson:"total_uber_duration"`
	Total_distance float64 `json:"total_distance" bson:"total_distance"`
	Uber_wait_time_eta float64 `json:"uber_wait_time_eta,omitempty"`
	Index int `json:"-"`

}

func PostTriplocation(rw http.ResponseWriter, request *http.Request,p httprouter.Params){
	var idResult DataId
	var input Request
	var data Database
	var output Response
	var total float64
	var sortSlice [] OrderedSlice
	req,_:= ioutil.ReadAll(request.Body)
	json.Unmarshal(req,&input)
	session, err := mgo.Dial("mongodb://user:pass@ds047901.mongolab.com:47901/cmpe273")
	if(err!=nil){
		defer session.Close()
	}
	c := session.DB("cmpe273").C("users")
	oid := bson.ObjectIdHex(input.Starting_from_location_id)
	err = c.FindId(oid).One(&data)
	if(err!=nil){
		log.Printf("RunQuery : ERROR : %s\n", err) 
		fmt.Fprintln(rw,err)
				return
	}
	start_latitude:= strconv.FormatFloat(data.Coor.Lat,'f',-1,64)
	start_longitude:= strconv.FormatFloat(data.Coor.Lng,'f',-1,64)
	var end_latitude,end_longitude string
	output.Starting_from_location_id = input.Starting_from_location_id
	var jsonInt interface{}
	sortSlice= make([]OrderedSlice,len(input.Location_ids))
	
	for p, i:= range input.Location_ids{
		oid := bson.ObjectIdHex(i)
		err = c.FindId(oid).One(&data)
		if(err!=nil){
		log.Printf("RunQuery : ERROR : %s\n", err) 
		fmt.Fprintln(rw,err)
				return
		}else{
			end_latitude = strconv.FormatFloat(data.Coor.Lat,'f',-1,64)
			end_longitude = strconv.FormatFloat(data.Coor.Lng,'f',-1,64)
			response,err:= http.Get("https://sandbox-api.uber.com/v1/estimates/price?start_latitude="+start_latitude+"&start_longitude="+start_longitude+"&end_latitude="+end_latitude+"&end_longitude="+end_longitude+"&access_token="+key+"")
			if err!=nil{
				fmt.Println("Error:",err)
			}else{
				defer response.Body.Close()
				contents,err:= ioutil.ReadAll(response.Body)
				if(err!=nil){
					fmt.Println(err)
				}
				json.Unmarshal(contents,&jsonInt)
				sortSlice[p].Cost = ((jsonInt.(map[string]interface{})["prices"]).([]interface{})[0].(map[string]interface{})["low_estimate"]).(float64)
				sortSlice[p].Duration =  ((jsonInt.(map[string]interface{})["prices"]).([]interface{})[0].(map[string]interface{})["duration"]).(float64)
				sortSlice[p].Distance = sortSlice[p].Distance + ((jsonInt.(map[string]interface{})["prices"]).([]interface{})[0].(map[string]interface{})["distance"]).(float64)		
				total = sortSlice[p].Cost * sortSlice[p].Duration
				sortSlice[p].Location_id = i
				sortSlice[p].Total = total
			}
		}
	}
	sort.Sort(ByTotal(sortSlice))
	output.Best_route_location_ids=make([]string,len(input.Location_ids))
	output.Best_route_location_ids[0]=sortSlice[0].Location_id
	output.Total_uber_costs = sortSlice[0].Cost
	output.Total_uber_duration =sortSlice[0].Duration
	output.Total_distance=sortSlice[0].Distance
	output.Index=0
	Array:=make([]string,len(sortSlice)) 
	for a,arr:= range sortSlice{
		Array[a] = arr.Location_id
	}
	temp:=1
	length:=len(Array)
	if length > 1{
		for j:=1; j < length ;j++{
			 sortSlice = Sorting(Array,Array[0])
			 if(len(sortSlice)!=0){
				 output.Best_route_location_ids[j] =sortSlice[0].Location_id
				 output.Total_uber_costs = output.Total_uber_costs + sortSlice[0].Cost
				 output.Total_uber_duration = output.Total_uber_duration + sortSlice[0].Duration
				 output.Total_distance = output.Total_distance+ sortSlice[0].Distance
				 for z:=1;z<len(Array);z++{
					Array[z] = sortSlice[z-1].Location_id
					}
			 }else{
			 	output.Best_route_location_ids[j]=Array[0]
			 }
			 if(len(Array) > 1){
			 	Array = Array[temp:]
			 }
		}
	}
	tempArray :=[] string{Array[0],input.Starting_from_location_id}
	sortSlice = Sorting(tempArray,Array[0])
	output.Total_uber_costs = output.Total_uber_costs + sortSlice[0].Cost
	output.Total_uber_duration = output.Total_uber_duration + sortSlice[0].Duration
	output.Total_distance = output.Total_distance+ sortSlice[0].Distance
	output.Status="planning"
	o := session.DB("cmpe273").C("trips")
	idResult.Id = 0
	count,_:=o.Count()
	if(count > 0){
		err := o.Find(nil).Select(bson.M{"id":100}).Sort("-id").One(&idResult)
		if(err!=nil){
			log.Printf("RunQuery : ERROR : %s\n", err) 
			fmt.Fprintln(rw,err)
			return 
		}
		output.Id = idResult.Id + 100
        err = o.Insert(output)
        if err != nil {
                log.Fatal(err)
        }
        result,_:=json.Marshal(output)
		fmt.Fprintln(rw,string(result))
	}else{
		output.Id = idResult.Id + 100
        err = o.Insert(output)
        if err != nil {
                log.Fatal(err)
        }
        result,_:=json.Marshal(output)
		fmt.Fprintln(rw,string(result))	
	}
}

func GetTripulocation(rw http.ResponseWriter, request *http.Request,p httprouter.Params){
	params,_:= strconv.Atoi(p.ByName("tripid"))
	session, err := mgo.Dial("mongodb://user:pass@ds047901.mongolab.com:47901/cmpe273")
	if(err!=nil){
		defer session.Close()
	}
	c := session.DB("cmpe273").C("trips")
	var data Response
	err = c.Find(bson.M{"id":params}).Select(bson.M{"_id":0}).One(&data)
	if(err!=nil){
		log.Printf("RunQuery : ERROR : %s\n", err) 
		fmt.Fprintln(rw,err)
				return
	}else{
		result,_:=json.Marshal(data)
		fmt.Fprintln(rw,string(result))	
	}
}

func Sorting(Array [] string,Starting_from_location_id string) []OrderedSlice{
	var data Database
	var total float64
	sortSlice:=make([] OrderedSlice,len(Array)-1)
	session, err := mgo.Dial("mongodb://user:pass@ds047901.mongolab.com:47901/cmpe273")
	if(err!=nil){
		defer session.Close()
	}
	c := session.DB("cmpe273").C("users")
	oid := bson.ObjectIdHex(Starting_from_location_id)
	err = c.FindId(oid).One(&data)
	if(err!=nil){
		log.Printf("RunQuery : ERROR : %s\n", err) 
	}
	start_latitude:= strconv.FormatFloat(data.Coor.Lat,'f',-1,64)
	start_longitude:= strconv.FormatFloat(data.Coor.Lng,'f',-1,64)
	var end_latitude,end_longitude string
	var jsonInt interface{}
	
	for p := 1;p < len(Array);p++{
		oid := bson.ObjectIdHex(Array[p])
		err = c.FindId(oid).One(&data)
		if(err!=nil){
		log.Printf("RunQuery : ERROR : %s\n", err) 
		}else{
			end_latitude = strconv.FormatFloat(data.Coor.Lat,'f',-1,64)
			end_longitude = strconv.FormatFloat(data.Coor.Lng,'f',-1,64)
			response,err:= http.Get("https://sandbox-api.uber.com/v1/estimates/price?start_latitude="+start_latitude+"&start_longitude="+start_longitude+"&end_latitude="+end_latitude+"&end_longitude="+end_longitude+"&access_token="+key+"")
			if err!=nil{
				fmt.Println("Error:",err)
			}else{
				defer response.Body.Close()
				contents,err:= ioutil.ReadAll(response.Body)
				if(err!=nil){
					fmt.Println(err)
				}
				json.Unmarshal(contents,&jsonInt)
				sortSlice[p-1].Cost = ((jsonInt.(map[string]interface{})["prices"]).([]interface{})[0].(map[string]interface{})["low_estimate"]).(float64)
				sortSlice[p-1].Duration =  ((jsonInt.(map[string]interface{})["prices"]).([]interface{})[0].(map[string]interface{})["duration"]).(float64)			
				sortSlice[p-1].Distance = ((jsonInt.(map[string]interface{})["prices"]).([]interface{})[0].(map[string]interface{})["distance"]).(float64)		
				total = sortSlice[p-1].Cost * sortSlice[p-1].Duration
				sortSlice[p-1].Location_id = Array[p]
				sortSlice[p-1].Total = total
				
			}
		}
	}
	sort.Sort(ByTotal(sortSlice))
	return sortSlice
}

func PutTriplocation(rw http.ResponseWriter, request *http.Request,p httprouter.Params){
	var request_id string
	var eta float64
	var status string
	var m Message
	params,_:= strconv.Atoi(p.ByName("tripid"))	
	session, err := mgo.Dial("mongodb://user:pass@ds047901.mongolab.com:47901/cmpe273")
	if(err!=nil){
		defer session.Close()
	}
	c := session.DB("cmpe273").C("trips")
	var data Response
	err = c.Find(bson.M{"id":params}).Select(bson.M{"_id":0}).One(&data)
	if(err!=nil){
		log.Printf("RunQuery : ERROR : %s\n", err) 
		fmt.Fprintln(rw,err)
				return
	}
	index:= data.Index
	var tempIn string
	if(index==0){
		tempIn = data.Starting_from_location_id
	}else{
		tempIn = data.Best_route_location_ids[index-1]
	}
	if(index <len(data.Best_route_location_ids)){
		if(data.Starting_from_location_id!=data.Best_route_location_ids[index]){
			request_id,eta,status = getDetails(tempIn,data.Best_route_location_ids[index])
			data.Next_destination_location_id = data.Best_route_location_ids[index]
			data.Uber_wait_time_eta = eta
			data.Status = status
			jsonStr,_:= json.Marshal(map[string] interface{}{"status":"completed"})
			req,err := http.NewRequest("PUT","https://sandbox-api.uber.com/v1/sandbox/requests/"+request_id,bytes.NewBuffer(jsonStr))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization","Bearer "+key)
			client := &http.Client{}
		    resp, err := client.Do(req)
		    if err!=nil{
				fmt.Println("Error:",err)
			}else{
				defer resp.Body.Close()
				result,_:=json.Marshal(data)
				fmt.Fprintln(rw,string(result))	
				tempIn = data.Best_route_location_ids[index]
				index++
				err = c.Update(bson.M{"id":params},bson.M{"$set":bson.M{"index":index}})
				
			}	
		}else{
			m.Msg = "Starting Location and destination cannot be same. Place another PUT"
			result,_:=json.Marshal(m)
			fmt.Fprintln(rw,string(result))
			index++
			err = c.Update(bson.M{"id":params},bson.M{"$set":bson.M{"index":index}})
			}
	}else{
			request_id,eta,status = getDetails(tempIn,data.Starting_from_location_id)
			data.Next_destination_location_id = data.Starting_from_location_id
			data.Uber_wait_time_eta = eta
			data.Status = status
			jsonStr,_:= json.Marshal(map[string] interface{}{"status":"completed"})
			req,err := http.NewRequest("PUT","https://sandbox-api.uber.com/v1/sandbox/requests/"+request_id,bytes.NewBuffer(jsonStr))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization","Bearer "+key)
			client := &http.Client{}
		    resp, err := client.Do(req)
		    if err!=nil{
				fmt.Println("Error:",err)
			}else{
				defer resp.Body.Close()
				result,_:=json.Marshal(data)
				fmt.Fprintln(rw,string(result))	
				index=0
				err = c.Update(bson.M{"id":params},bson.M{"$set":bson.M{"index":index}})
				
			}
	}

}

func getDetails(start string, end string)(string,float64,string){
	var jsonInt interface{}
	var data Database
	session, err := mgo.Dial("mongodb://user:pass@ds047901.mongolab.com:47901/cmpe273")
	if(err!=nil){
		defer session.Close()
	}
	c := session.DB("cmpe273").C("users")
	oid := bson.ObjectIdHex(start)
    err = c.FindId(oid).One(&data)
	if(err!=nil){
		log.Printf("RunQuery : ERROR : %s\n", err) 
	}
	start_latitude:= strconv.FormatFloat(data.Coor.Lat,'f',-1,64)
	start_longitude:= strconv.FormatFloat(data.Coor.Lng,'f',-1,64)
	var end_latitude,end_longitude string
	oid1 := bson.ObjectIdHex(end)
	err = c.FindId(oid1).One(&data)
	if(err!=nil){
	log.Printf("RunQuery : ERROR : %s\n", err) 
	}else{
		end_latitude = strconv.FormatFloat(data.Coor.Lat,'f',-1,64)
		end_longitude = strconv.FormatFloat(data.Coor.Lng,'f',-1,64)
		product_id := GetProductId(start_latitude,start_longitude)
		jsonStr,_:= json.Marshal(map[string] interface{}{
		"product_id":product_id,"start_latitude":start_latitude,"start_longitude":start_longitude,"end_latitude":end_latitude,"end_longitude":end_longitude})
		req,err := http.NewRequest("POST","https://sandbox-api.uber.com/v1/requests",bytes.NewBuffer(jsonStr))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization","Bearer "+key)
		client := &http.Client{}
	    resp, err := client.Do(req)
	    if err!=nil{
			fmt.Println("Error:",err)
		}else{
			defer resp.Body.Close()
			contents,err:= ioutil.ReadAll(resp.Body)
			if(err!=nil){
				fmt.Println(err)
			}else{	
				json.Unmarshal(contents,&jsonInt)
				request_id:= jsonInt.(map[string]interface{})["request_id"]
				eta:= jsonInt.(map[string]interface{})["eta"]
				status:= jsonInt.(map[string]interface{})["status"]
				return request_id.(string),eta.(float64),status.(string)
			}
		}
	}
	return "",0.0,""
}

func GetProductId(start string, end string)(string){
	var jsonInt interface{}
	response,err:= http.Get("https://sandbox-api.uber.com/v1/products?latitude="+start+"&longitude="+end+"&access_token="+key+"")
	if err!=nil{
		fmt.Println("Error:",err)
	}else{
		defer response.Body.Close()
		contents,err:= ioutil.ReadAll(response.Body)
		if(err!=nil){
			fmt.Println(err)
		}
		json.Unmarshal(contents,&jsonInt)
		product_id:= (jsonInt.(map[string] interface{})["products"]).([]interface{})[0].(map[string]interface{})["product_id"]
		return product_id.(string)
	}
	return ""
}
