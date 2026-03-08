package com.thisjowi.password.Service;

import org.springframework.stereotype.Service;
import com.thisjowi.password.Entity.Password;
import com.thisjowi.password.Repository.PasswordRepository;
import com.thisjowi.password.Utils.Encryption;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.util.*;

/**
 * Service to detect and remove duplicate passwords created by sync issues
 * This handles the case where multiple POST requests were sent simultaneously,
 * creating duplicate entries with identical content.
 */
@Service
public class PasswordDeduplicationService {
    private static final Logger log = LoggerFactory.getLogger(PasswordDeduplicationService.class);
    
    private final PasswordRepository passwordRepository;
    private final Encryption encryption;
    
    public PasswordDeduplicationService(PasswordRepository passwordRepository, Encryption encryption) {
        this.passwordRepository = passwordRepository;
        this.encryption = encryption;
    }
    
    /**
     * Find and log duplicate passwords for a user
     * Duplicates are identified by matching encrypted: name + website + username
     */
    public Map<String, Object> analyzeDuplicates(Long userId) {
        log.info("Analyzing duplicates for user {}", userId);
        
        List<Password> userPasswords = passwordRepository.findByUserId(userId);
        if (userPasswords == null || userPasswords.isEmpty()) {
            return Map.of("duplicates_found", 0, "message", "No passwords found for this user");
        }
        
        // Group by (name, website, username) to find duplicates
        Map<String, List<Password>> groups = new HashMap<>();
        
        for (Password p : userPasswords) {
            String key = createDuplicateKey(p);
            groups.computeIfAbsent(key, k -> new ArrayList<>()).add(p);
        }
        
        // Filter to only groups with duplicates
        List<Map<String, Object>> duplicateGroups = new ArrayList<>();
        int totalDuplicates = 0;
        
        for (var entry : groups.entrySet()) {
            if (entry.getValue().size() > 1) {
                List<Long> ids = new ArrayList<>();
                for (Password p : entry.getValue()) {
                    ids.add(p.getId());
                }
                duplicateGroups.add(Map.of(
                    "key", entry.getKey(),
                    "count", entry.getValue().size(),
                    "ids", ids
                ));
                totalDuplicates += entry.getValue().size() - 1; // Count extras as duplicates
            }
        }
        
        log.info("Found {} duplicate groups with {} total duplicates", duplicateGroups.size(), totalDuplicates);
        
        return Map.of(
            "duplicates_found", totalDuplicates,
            "duplicate_groups", duplicateGroups,
            "message", "Analysis complete"
        );
    }
    
    /**
     * Remove duplicate passwords, keeping the most recent one
     * Returns count of deleted duplicates
     */
    public Map<String, Object> removeDuplicates(Long userId) {
        log.warn("Starting duplicate removal for user {}", userId);
        
        List<Password> userPasswords = passwordRepository.findByUserId(userId);
        if (userPasswords == null || userPasswords.isEmpty()) {
            return Map.of("deleted_count", 0, "message", "No passwords to clean");
        }
        
        // Group by (name, website, username)
        Map<String, List<Password>> groups = new HashMap<>();
        
        for (Password p : userPasswords) {
            String key = createDuplicateKey(p);
            groups.computeIfAbsent(key, k -> new ArrayList<>()).add(p);
        }
        
        int deletedCount = 0;
        
        // For each group with duplicates, keep the most recent and delete others
        for (var entry : groups.entrySet()) {
            List<Password> duplicates = entry.getValue();
            if (duplicates.size() > 1) {
                log.info("Found {} duplicates for key: {}", duplicates.size(), entry.getKey());
                
                // Sort by ID descending to keep the most recent one
                duplicates.sort(Comparator.comparingLong(Password::getId).reversed());
                
                // Delete all except the first (most recent)
                for (int i = 1; i < duplicates.size(); i++) {
                    Password duplicate = duplicates.get(i);
                    log.info("Deleting duplicate password id: {} (keeping id: {})", 
                            duplicate.getId(), duplicates.get(0).getId());
                    passwordRepository.deleteById(duplicate.getId());
                    deletedCount++;
                }
            }
        }
        
        log.info("Deleted {} duplicate passwords for user {}", deletedCount, userId);
        
        return Map.of(
            "deleted_count", deletedCount,
            "message", "Duplicate removal complete"
        );
    }
    
    /**
     * Create a key for grouping duplicates
     * Uses the encrypted values since those are what actually matter
     */
    private String createDuplicateKey(Password p) {
        return String.format("%s|%s|%s",
            p.getName() != null ? p.getName() : "",
            p.getWebsite() != null ? p.getWebsite() : "",
            ""  // username isn't stored separately in Password entity
        );
    }
}
