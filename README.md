# thermostat-project

This is a representation of a REST API used to control and obtain information regarding x number of thermostats in a home.
There are two directories:

  - <b>apidocs</b>
      - contains swagger api documentation to provide a high level overview of all endpoints
      - to view the API documentation
          - <i>cd apidocs</i>
          - <i>python -m SimpleHTTPServer 8000</i>
          - navigate to <a href="http://localhost:8000">http://localhost:8000</a>
  - <b>server</b>
      - contains the executable API server and a full unit test suite to validate all endpoints included in the API
      - to run the test suite
          - <i>cd server</i>
          - <i>go test -v</i>
      - to run the web server
          - <i>cd server</i>
          - <i>go run server</i>
