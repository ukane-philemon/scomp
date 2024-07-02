# SCOMP 📟 
SCOMP is a Graphql-based computational service that computes student
records in a class and grades each student according to their respective performance in reported
subjects. 

## Features ⚡
1. Create an admin account.
2. Login to an existing admin account.
3. Create a class.
4. Add a student record to an existing class.
5. Compute class report.
6. Query class record.
7. Query student record.
8. Query all existing classes.

## Limitations ⚠️

1. A class must contain at least 2 students to compute class reports.
2. Class report can only be generated once.
3. Student records cannot be added to a class after a report has been generated
   for that class.
4. To use this service for the same class with another set of students, classes
should be created with names formatted like `ClassName Year`, e.g `JSS1 2024`
etc. Then students for the newly created class should be added (minimum of 2).
5. Admin would need to request for class `report` after a delay. This is because
   report computation is done asynchronously.
6. Students with the same score will have different class position.
7. Students with the same subject score will have different position.

## Starting the Server: Perquisites 💻

1. Go installed.
2. A database connection URL from mongodb.com


## How to start the application server 🚀

1. Ensure the latest version of `go` is installed on your device. Visit
   https://go.dev/doc/install to install `go`.

2. Clone this repo to your local device and run `cd scomp` on your terminal.

3. Set value for environment variable `DB_URL` {required} and `PORT` {optional, default: `8080`}.

3. Lastly, run `go build` to build the executable and then run `./scomp --dev {remove --dev for production}` to start the HTTP server.

4. Visit `localhost:PORT` to view the Graphql playground.

## Documentation

Graphql Playground: https://scomp.onrender.com/

OR

Visit the [API Documentation on Postman](https://www.postman.com/fewchore-api/workspace/meg/collection/668324efe630760afe868062?action=share&source=copy-link&creator=9797704)