# external-power-timer
 
# External Power Timer

External Power Timer is a Windows-based application designed to track the runtime of an electric generator. It provides a simple API interface to control a timer window, which remains hidden until activated via API calls. The timer window can be moved but not closed manually, ensuring accurate tracking of the generator's operation time.

## Features

- Hidden Background Process: Runs silently until triggered.

- Timer Display: Shows a movable but non-closable timer.

## Simple API Control:

- POST /create - Displays the timer and starts counting.

- POST /reset - Resets the timer without hiding it.

- POST /close - Hides and resets the timer, waiting for another activation.

## Installation

- cd external-power-timer

- Initialize a Go module:

go mod init external-power-timer

- Install dependencies:

go get github.com/lxn/walk
go get github.com/lxn/win

- Build the executable:

go build -o timer.exe

## Usage

Run the application:

timer.exe

Use an API client (e.g., Postman, curl) to control the timer.

Example API Calls

Start the timer:

curl -X POST http://localhost:1997/create

Reset the timer:

curl -X POST http://localhost:1997/reset

Hide and reset the timer:

curl -X POST http://localhost:1997/close

Show the timer and start counting
Invoke-RestMethod -Uri "http://localhost:1997/create" -Method Post

Reset the timer without closing the window
Invoke-RestMethod -Uri "http://localhost:1997/reset" -Method Post

Hide and reset the timer (waiting for another "create" call)
Invoke-RestMethod -Uri "http://localhost:1997/close" -Method Post
