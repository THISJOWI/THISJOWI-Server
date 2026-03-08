package com.thisjowi.note.service;

import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.http.HttpHeaders;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.stereotype.Service;
import org.springframework.web.reactive.function.client.ClientResponse;
import org.springframework.web.reactive.function.client.WebClient;
import reactor.core.publisher.Mono;
import java.util.Map;

@Service
public class AuthenticationClient {

    private final WebClient authenticationWebClient;
    private final Logger log = LoggerFactory.getLogger(AuthenticationClient.class);

    @Autowired
    public AuthenticationClient(WebClient authenticationWebClient) {
        this.authenticationWebClient = authenticationWebClient;
    }

    public boolean validateToken(String token) {
        // The Authentication service doesn't expose a /validate endpoint.
        // Reuse getUserIdFromToken: if it returns a valid id (>=0) the token is valid.
        Long id = getUserIdFromToken(token);
        return id != null && id != -1L;
    }

    public Long getUserIdFromToken(String token) {
        String headerValue = token != null && token.startsWith("Bearer ") ? token : ("Bearer " + token);
        log.debug("Calling Authentication service /user to validate token");

        Mono<Long> mono = authenticationWebClient.get()
                .uri(uriBuilder -> uriBuilder.path("/v1/auth/user").build())
                .header(HttpHeaders.AUTHORIZATION, headerValue)
                .exchangeToMono((ClientResponse resp) -> {
                    if (resp.statusCode().is2xxSuccessful()) {
                        return resp.bodyToMono(Map.class)
                                .map(body -> {
                                    Object userIdObj = body.get("userId");
                                    if (userIdObj instanceof Number) {
                                        return ((Number) userIdObj).longValue();
                                    }
                                    log.warn("Auth service returned body without valid userId: {}", body);
                                    return -1L;
                                });
                    } else {
                        return resp.bodyToMono(String.class)
                                .defaultIfEmpty("(no body)")
                                .flatMap(body -> {
                                    log.warn("Auth service returned {} with body: {}", resp.statusCode().value(), body);
                                    return Mono.just(-1L);
                                });
                    }
                })
                .onErrorResume(e -> {
                    log.error("Error calling auth service to get user id", e);
                    return Mono.just(-1L);
                });

        try {
            return mono.block();
        } catch (Exception e) {
            log.error("Error blocking for user id", e);
            return -1L;
        }
    }
}