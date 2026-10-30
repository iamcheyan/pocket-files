package main

import (
 "database/sql"
 "encoding/json"
 "log"
 "net/http"
 "os"
 "strconv"
 "strings"
 _ "modernc.org/sqlite"
)

type Server struct { p *MockProvider; db *sql.DB }
type storageDTO struct { ID string `json:"id"`; Name string `json:"name"`; Kind string `json:"kind"`; Status string `json:"status"`; Mock bool `json:"mock"`; ReadOnly bool `json:"read_only"`; Total int64 `json:"total"`; Used int64 `json:"used"` }
func storageItems(devices []Device) []storageDTO { out:=make([]storageDTO,0,len(devices));for _,d:=range devices{out=append(out,storageDTO{ID:d.ID,Name:d.Name,Kind:d.Kind,Status:d.Status,Mock:d.Mock,ReadOnly:d.ReadOnly,Total:d.Total,Used:d.Used})};return out }
func entryItems(entries []Entry) []map[string]any { out:=make([]map[string]any,0,len(entries));for _,e:=range entries{out=append(out,map[string]any{"id":e.ID,"storage_id":e.StorageID,"name":e.Name,"parent":e.Parent,"kind":e.Kind,"size":e.Size,"modified":e.Modified,"extension":e.Extension})};return out }
func (s *Server) json(w http.ResponseWriter,status int,v any) { w.Header().Set("Content-Type","application/json");w.WriteHeader(status);_ = json.NewEncoder(w).Encode(v) }
func (s *Server) error(w http.ResponseWriter,status int,code,msg string) { s.json(w,status,map[string]any{"error":map[string]string{"code":code,"message":msg}}) }
func (s *Server) ServeHTTP(w http.ResponseWriter,r *http.Request) {
 path:=strings.Trim(r.URL.Path,"/");parts:=strings.Split(path,"/")
 if path=="api/health" { s.json(w,200,map[string]any{"status":"ok","mode":"mock"});return }
 if len(parts)>=2&&parts[0]=="api" { switch parts[1] {
 case "storage": if r.Method=="GET" { d,_:=s.p.Discover(r.Context());s.json(w,200,map[string]any{"items":storageItems(d)});return }
 case "files": if len(parts)==2&&r.Method=="GET" { id:=r.URL.Query().Get("storage_id");q:=r.URL.Query().Get("q");var items []Entry;var e error;if q!=""{items,e=s.p.Search(r.Context(),id,q)}else{items,e=s.p.Browse(r.Context(),id,r.URL.Query().Get("path"))};if e!=nil{s.error(w,400,"storage_error",e.Error());return};if ext:=r.URL.Query().Get("ext");ext!=""{filtered:=items[:0];for _,x:=range items{if strings.EqualFold(x.Extension,strings.TrimPrefix(ext,".")){filtered=append(filtered,x)}};items=filtered};limit,_:=strconv.Atoi(r.URL.Query().Get("limit"));if limit<=0||limit>500{limit=100};page,_:=strconv.Atoi(r.URL.Query().Get("page"));start:=page*limit;if start>len(items){start=len(items)};items=items[start:];if len(items)>limit{items=items[:limit]};s.json(w,200,map[string]any{"items":entryItems(items),"page":page,"limit":limit});return }
 case "folders": if r.Method=="POST" { var v struct{StorageID string `json:"storage_id"`;Path string `json:"path"`};if json.NewDecoder(r.Body).Decode(&v)!=nil{s.error(w,400,"invalid_json","invalid request");return};if e:=s.p.CreateDir(r.Context(),v.StorageID,v.Path);e!=nil{s.error(w,400,"operation_failed",e.Error());return};w.WriteHeader(201);return }
 case "favorites": if r.Method=="GET" { s.json(w,200,map[string]any{"items":[]any{}});return }
 case "recents": if r.Method=="GET" { s.json(w,200,map[string]any{"items":[]any{}});return }
 case "views": if len(parts)==3 { s.error(w,501,"not_implemented","virtual view is not available for this provider yet");return }
 case "downloads": s.json(w,503,map[string]any{"status":"unavailable","source":"downloadsd"});return
 case "settings": s.json(w,200,map[string]any{"mode":"mock","privacy_mode":false});return
 case "transfers": s.json(w,200,map[string]any{"items":[]any{}});return
 } }
 s.error(w,404,"not_found","endpoint not found")
}
func openDB(path string)(*sql.DB,error){db,e:=sql.Open("sqlite",path);if e!=nil{return nil,e};queries:=[]string{`PRAGMA foreign_keys=ON`,`CREATE TABLE IF NOT EXISTS schema_version(version INTEGER NOT NULL)`,`CREATE TABLE IF NOT EXISTS storage_devices(id TEXT PRIMARY KEY,friendly_name TEXT NOT NULL,kind TEXT NOT NULL,status TEXT NOT NULL,mock INTEGER NOT NULL DEFAULT 0)`,`CREATE TABLE IF NOT EXISTS directories(id TEXT PRIMARY KEY,storage_id TEXT NOT NULL,parent_id TEXT,name TEXT NOT NULL)`,`CREATE TABLE IF NOT EXISTS favorites(id INTEGER PRIMARY KEY,storage_id TEXT NOT NULL,uri TEXT NOT NULL,UNIQUE(storage_id,uri))`,`CREATE TABLE IF NOT EXISTS recent_files(id INTEGER PRIMARY KEY,storage_id TEXT NOT NULL,uri TEXT NOT NULL,kind TEXT NOT NULL,accessed_at TEXT NOT NULL)`,`CREATE TABLE IF NOT EXISTS transfer_jobs(id TEXT PRIMARY KEY,status TEXT NOT NULL,payload TEXT NOT NULL,updated_at TEXT NOT NULL)`,`CREATE TABLE IF NOT EXISTS settings(key TEXT PRIMARY KEY,value TEXT NOT NULL)`,`CREATE TABLE IF NOT EXISTS file_tags(id INTEGER PRIMARY KEY,storage_id TEXT NOT NULL,uri TEXT NOT NULL,tag TEXT NOT NULL)`,`CREATE TABLE IF NOT EXISTS custom_locations(id INTEGER PRIMARY KEY,name TEXT NOT NULL,storage_id TEXT NOT NULL,uri TEXT NOT NULL)`};for _,q:=range queries{if _,e=db.Exec(q);e!=nil{db.Close();return nil,e}};return db,nil}
func main(){p,e:=newMockProvider();if e!=nil{log.Fatal(e)};db,e:=openDB(env("POCKET_FILES_DB","pocket-files.db"));if e!=nil{log.Fatal(e)};defer db.Close();addr:=env("POCKET_FILES_ADDR","127.0.0.1:8787");log.Printf("filesd listening on %s (mock mode)",addr);if e=http.ListenAndServe(addr,&Server{p:p,db:db});e!=nil{log.Fatal(e)}}
func env(k,d string)string{if v:=os.Getenv(k);v!=""{return v};return d}
