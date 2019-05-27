# To setup for local development and pushing updates follow these steps
## Clone the code
- Clone our github repo at https://github.com/seaptc/seaptc, Gary or Alfred are currently admins and can help
## Setup GCloud
- Get permissions to our gcloud instance (https://console.cloud.google.com/home/dashboard?project=seaptc-ds), Gary or Alfred are currently admins and can help
- Install the gcloud runtime on your machine ( https://cloud.google.com/appengine/docs/standard/python/download )
- Make sure you are on the right gcloud project (if you didn't just install from scratch):  
```
gcloud config set project seaptc-ds
```
- Setup the default login to the account you had been given access on:
```
gcloud auth application-default login
```
## Install Go
- Install the Go runtime (https://golang.org/doc/install
## Configure your local instance
- Run a local datastore to host your instance:  
```
gcloud beta emulators datastore start
```
- Export your DS environment variable to other command prompts you run, NOTE the value here might change, look at the output from the previous command:  
```
export DATASTORE_EMULATOR_HOST=localhost:8081
```
- Run the copyprod.sh script after setting the local default login details:  
    ```
    cd <repo root>/server/store  
    ./copyprod.sh
    ```
## Run the local server
- Run the seaptc server locally:  
    ```
    cd <repo root>/server  
    go install ./seaptc
    ~/go/bin/seaptc
    ```
- Navigate to http://localhost:8080/dashboard/admin in your browser of choice and login
## Iterate
- Write code, be merry!!
## Commit to production
- Once you are happy with local edits commit to git and push
- To deploy to the production system run:  
    ```
    cd <repo root>/server  
    gcloud app deploy
    ```