package com.thisjowi.password.Repository;

import org.springframework.data.jpa.repository.JpaRepository;
import org.springframework.data.jpa.repository.Query;
import org.springframework.data.repository.query.Param;
import com.thisjowi.password.Entity.Password;

import java.util.List;
import java.util.Optional;

public interface PasswordRepository extends JpaRepository<Password, Long> {

    List<Password> findByName(String name);
    
    List<Password> findByUserId(Long userId);
    
    /**
     * Find password by userId, name (title), and website
     * Used to detect duplicates
     */
    @Query("SELECT p FROM Password p WHERE p.userId = :userId AND p.name = :name AND p.website = :website")
    Optional<Password> findByUserIdAndNameAndWebsite(
        @Param("userId") Long userId,
        @Param("name") String name,
        @Param("website") String website
    );
}
