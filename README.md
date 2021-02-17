# Examples of MTLS on Applications

This repository contains 2 simple applications (REST microservices in Java and Go) used to explain the implementation of Mutual TLS Authentication.

| Application             | Description
| ---                     | ---         
| 1-greeting-java         | Simple REST microservice (Maven Project) based on Spring Boot 2.4.2 and Java 11. 
| 2-two-uservices-go      | Two Go microservices.
| 3-mtls-emojivoto-tf     | Example enabling MTLS on Buoyant [Emojivoto](https://github.com/buoyantio/emojivoto) microservices application using [SmallStep CA](https://github.com/smallstep/certificates) and [Envoy Proxy](https://www.envoyproxy.io/). This version is based on [Step AWS Emojivoto](https://github.com/smallstep/step-aws-emojivoto) example and it's adapted to be used for multiple users simultaneously without collisions.

## Preparation

Clone the repository.
```sh
## clone the repo
$ git clone https://github.com/chilcano/mtls-apps-examples
```

Install Java 11, Maven and Go.
```sh
$ source <(curl -s https://raw.githubusercontent.com/chilcano/how-tos/master/src/devops_playground_tools_install.sh) 
```

## Getting started with the 'Java Greeting REST service'

### Saying greeting (without encryption in transit)

1. Check the REST service (`src/main/resources/application.yml`) configuration. You should have this configuration:  
```yaml
server:
  port: 9090
``` 

2. Clean and build the project for the first time.
```sh
$ cd mtls-apps-examples/1-greeting-java 
$ mvn clean
$ mvn spring-boot:run
``` 

3. Calling the REST service.  
```sh
$ curl -i http://localhost:9090/greeting
```

4. It should give you the following response:  
```sh
HTTP/1.1 200 
Content-Type: application/json
Transfer-Encoding: chunked
Date: Tue, 16 Feb 2021 13:41:15 GMT

{"id":1,"content":"Hello, World!"}
```

5. Close the running REST service typing `Ctrl + C`.   

### Enabling HTTP over TLS (HTTPS) on the service (One-way TLS)

1. Update the `src/main/resources/application.yml` to enable One-way TLS.   
```yaml
server:
  port: 9443
  ssl:
    enabled: true
``` 

2. Restart the REST service so that it can apply the changes and test it.   
```sh
$ mvn clean spring-boot:run
$ curl -i https://localhost:9443/greeting
``` 

We will probably get the following error:
```sh
Caused by: java.lang.IllegalArgumentException: Resource location must not be null
        at org.springframework.util.Assert.notNull(Assert.java:201) ~[spring-core-5.3.3.jar:5.3.3]
        at org.springframework.util.ResourceUtils.getURL(ResourceUtils.java:130) ~[spring-core-5.3.3.jar:5.3.3]
        at org.springframework.boot.web.embedded.tomcat.SslConnectorCustomizer.configureSslKeyStore(SslConnectorCustomizer.java:129) ~[spring-boot-2.4.2.jar:2.4.2]
        ... 16 common frames omitted
```

We are getting this message because the REST service (server) requires a keystore with the certificate of the REST service (server) to ensure that there is a secure connection with the outside world.  
To solve this, we are going to create a keystore with a public and private key for the REST service (server). The public key will be shared with users/clients so that they can encrypt the communication. 
The communication between both parties (user and server) can be decrypted with the private key of the REST service (server). 
The private key of the REST service (server) never must be shared and must be keep it secret, symmetrically encrypted or in a vault (i.e. PKCS#7, HSM, Hashicorp Vault).

3. Generate the server certificate.   

Any Java application use [keystore](https://en.wikipedia.org/wiki/Java_KeyStore) file as repository of public-key certificates and asymmetric private keys. Then, to create a keystore with a public and private key, execute the following command in your terminal:
```sh
$ keytool -v \
        -genkeypair \
        -dname "CN=Server (MTLS for Java Microservice),OU=DevOps Playground,O=ECS,C=UK" \
        -keystore src/main/resources/server_identity.jks \
        -storepass secret \
        -keypass secret \
        -keyalg RSA \
        -keysize 2048 \
        -alias server \
        -validity 3650 \
        -deststoretype pkcs12 \
        -ext KeyUsage=digitalSignature,dataEncipherment,keyEncipherment,keyAgreement \
        -ext ExtendedKeyUsage=serverAuth,clientAuth \
        -ext SubjectAlternativeName:c=DNS:localhost,IP:127.0.0.1

Generating 2,048 bit RSA key pair and self-signed certificate (SHA256withRSA) with a validity of 3,650 days
        for: CN=MTLS for Java Microservice, OU=DevOps Playground, O=ECS, C=UK
[Storing src/main/resources/server_identity.jks]
```

Once generated the TLS certificate, you will need to update the REST service (server) `src/main/resources/application.yml` file with the location of the keystore and symmetric passwords required for keystore itself and for private key.  
```yaml
server:
  port: 9443
  ssl:
    enabled: true
    key-store: classpath:server_identity.jks
    key-password: secret
    key-store-password: secret
```

4. Test the One-way TLS connection.   
```sh
$ mvn clean spring-boot:run
$ curl -i --insecure -v https://localhost:9443/greeting

## alternatively with '-k' option
$ curl -i -k https://localhost:9443/greeting
``` 

If everything worked, then you should see this:
```sh
{"id":1,"content":"Hello, World!"}
```

Now, if we remove the `--insecure` or `-k` we will get this error:
```sh
$ curl -i https://localhost:9443/greeting

curl: (60) SSL certificate problem: unable to get local issuer certificate
More details here: https://curl.haxx.se/docs/sslcerts.html

curl failed to verify the legitimacy of the server and therefore could not
establish a secure connection to it. To learn more about this situation and
how to fix it, please visit the web page mentioned above.
```

That means `curl` (client) can not get validated the REST service's TLS certificate because the client don't have or don't trust the CA that issued the REST service certificate.
And if you open `https://localhost:9443/greeting` in your browser (another client) you will get similar error (see below image).

![](img/mtls-java-1-err-cert-authority-invalid.png)

To avoid this, you need to have the certificate(s) of the server and you can get it with the following command:
```sh
$ keytool -v \
    -exportcert \
    -file src/main/resources/server.crt \
    -alias server \
    -keystore src/main/resources/server_identity.jks \
    -storepass secret \
    -rfc 

Certificate stored in file <src/main/resources/server.crt>
```

Now, install `src/main/resources/server.crt` in your browser or use it with curl command to call the REST service.
```sh
$ curl -i --cacert src/main/resources/server.crt https://localhost:9443/greeting

HTTP/1.1 200 
Content-Type: application/json
Transfer-Encoding: chunked
Date: Tue, 16 Feb 2021 17:31:37 GMT

{"id":3,"content":"Hello, World!"}
```


### Enabling Mutual TLS Authentication (Two-way TLS)

The configuration of MTLS (Two-way TLS) in the server will require a new certificate for the authentication of the client. 
This configuration will force the client (curl, your browser or any proper HTTP client) to identify itself using a certificate, and in that way, the server (REST service) 
can also validate the identity of the client and whether or not it is a trusted one. 
You can get this by configuring the server (REST service) that you also want to validate the client with the property `client-auth` in the `src/main/resources/application.yml` file.   

1. Update following properties in the `src/main/resources/application.yml` file of the REST service (server):   
```yaml
server:
  port: 9443
  ssl:
    enabled: true
    key-store: classpath:server_identity.jks
    key-password: secret
    key-store-password: secret
    client-auth: need                ## require client authn
``` 

2. Run your client to check REST service MTLS configuration.   


```sh
$ curl -i --cacert src/main/resources/server.crt https://localhost:9443/greeting

curl: (56) OpenSSL SSL_read: error:14094412:SSL routines:ssl3_read_bytes:sslv3 alert bad certificate, errno 0
```

Running the curl client will fail with the following error message: `error:14094412:SSL routines:ssl3_read_bytes:sslv3 alert bad certificate, errno 0`. This indicates that the certificate of the client is not valid because there is no certificate at all. So, let's create one with the following command:

```sh
$ keytool -v \
        -genkeypair \
        -dname "CN=Client (MTLS for Java Microservice),OU=DevOps Playground,O=ECS,C=UK" \
        -keystore src/main/resources/client_identity.jks \
        -storepass secret \
        -keypass secret \
        -keyalg RSA \
        -keysize 2048 \
        -alias client \
        -validity 3650 \
        -deststoretype pkcs12 \
        -ext KeyUsage=digitalSignature,dataEncipherment,keyEncipherment,keyAgreement \
        -ext ExtendedKeyUsage=serverAuth,clientAuth 
```

> The above command will not add the `SubjectAlternativeName` attribute to the client certificate (`-ext SubjectAlternativeName:c=DNS:<client-fqdn>,IP:<client-ip-address>`) because the client (curl or browser) will be executed in the same host where the REST service is running. But if you want to execute the client (curl or browser) from different host, you could set a `SubjectAlternativeName` attribute with a `fqdn`, `hostname` or `IP address` what the REST service (server) can resolv and validate without issues.   
> You can simulate this behaviour when running the client and server in the same host, only you have to add as client's hostname and server's hostname to the `/etc/hosts` file.

Once the `client_identity.jks` (private key and public key certificate) has been generated, we must tell the server about which root and intermediate certificates to trust. This is done creating a `truststore` containing all those trusted certificates. We can get the client certificate extracting it from previously generated `client_identity.jks`.

3. Extract the client certificate from `client_identity.jks`

```sh
$ keytool -v \
        -exportcert \
        -file src/main/resources/client.crt \
        -alias client \
        -keystore src/main/resources/client_identity.jks \
        -storepass secret \
        -rfc 
```

4. Create the server truststore with the client certificate.  

```sh
$ keytool -v \
        -importcert \
        -file src/main/resources/client.crt \
        -alias client \
        -keystore src/main/resources/server_truststore.jks \
        -storepass secret \
        -noprompt
```





## References

### Go
* https://kofo.dev/how-to-mtls-in-golang
* https://venilnoronha.io/a-step-by-step-guide-to-mtls-in-go
* https://github.com/nicholasjackson/mtls-go-example
* https://smallstep.com/hello-mtls/doc/server/go

### Java
* https://spring.io/guides/gs/rest-service/