package com.thisjowi.note.service;

import org.springframework.stereotype.Service;
import org.springframework.transaction.annotation.Transactional;
import org.springframework.dao.DataIntegrityViolationException;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import com.thisjowi.note.Utils.EncryptionUtil;
import com.thisjowi.note.entity.Note;
import com.thisjowi.note.repository.NoteRepository;

import java.time.LocalDateTime;
import java.util.List;
import java.util.Optional;

@Service
public class NoteService {

    private static final Logger logger = LoggerFactory.getLogger(NoteService.class);
    private final NoteRepository noteRepository;

    public NoteService(NoteRepository noteRepository) {
        this.noteRepository = noteRepository;
    }

    // Get all notes (without filtering by user)
    @Transactional(readOnly = true)
    public List<Note> getAllNotes() {
        List<Note> notes = noteRepository.findAll();
        return notes.stream().map(this::decryptNote).toList();
    }

    // Search notes by title fragment (without filtering by user)
    @Transactional(readOnly = true)
    public List<Note> searchNotesByTitle(String title) {
        if (title == null) title = "";
        List<Note> notes = noteRepository.findByTitleIgnoreCaseContaining(title);
        return notes.stream().map(this::decryptNote).toList();
    }

    @Transactional
    public Note saveNote(Note note) {
        // Save original title before encryption for error handling
        String originalTitle = note.getTitle();
        
        note.setTitle(EncryptionUtil.encrypt(note.getTitle()));
        note.setContent(EncryptionUtil.encrypt(note.getContent()));
        
        try {
            Note saved = noteRepository.save(note);
            
            // Return a copy with decrypted content to avoid dirty checking update
            Note response = new Note();
            response.setId(saved.getId());
            response.setUserId(saved.getUserId());
            response.setCreatedAt(saved.getCreatedAt());
            response.setTitle(EncryptionUtil.decrypt(saved.getTitle()));
            response.setContent(EncryptionUtil.decrypt(saved.getContent()));
            return response;
        } catch (DataIntegrityViolationException e) {
            // Handle constraint violation: a note with same title for this user already exists
            // This can happen in concurrent scenarios - fetch and return the existing note
            logger.info("Constraint violation detected - note with title '{}' for user {} already exists", 
                       originalTitle, note.getUserId());
            
            Optional<Note> existing = noteRepository.findByTitleIgnoreCaseAndUserId(
                originalTitle, 
                note.getUserId()
            );
            
            if (existing.isPresent()) {
                return decryptNote(existing.get());
            } else {
                // This shouldn't happen, but re-throw if not found
                throw e;
            }
        }
    }

    /**
     * Save or update a note, preventing duplicates for the same user.
     * If a note with the same title already exists for the user, it will be updated.
     * Handles concurrent requests gracefully - if a constraint violation occurs,
     * the existing note is fetched and returned (or updated if content differs).
     */
    @Transactional
    public Note saveNoteWithDeduplication(Note note) {
        if (note.getUserId() == null) {
            throw new IllegalArgumentException("UserId is required");
        }

        Long userId = note.getUserId();
        String titleToCheck = note.getTitle() != null ? note.getTitle().trim() : "";

        if (titleToCheck.isEmpty()) {
            throw new IllegalArgumentException("Note title is required");
        }

        // Check if a note with the same title and user already exists
        Optional<Note> existingOptional = noteRepository.findByTitleIgnoreCaseAndUserId(titleToCheck, userId);

        if (existingOptional.isPresent()) {
            // Update existing note
            Note existing = existingOptional.get();
            Long existingId = existing.getId();
            logger.info("Duplicate note detected for user {}, updating existing note id: {}", userId, existingId);

            // Only update content if provided and different
            if (note.getContent() != null && !note.getContent().isEmpty()) {
                existing.setTitle(EncryptionUtil.encrypt(titleToCheck));
                existing.setContent(EncryptionUtil.encrypt(note.getContent()));
                existing.setUserId(userId);
                
                Note saved = noteRepository.save(existing);
                // Return with decrypted content
                Note response = new Note();
                response.setId(saved.getId());
                response.setUserId(saved.getUserId());
                response.setCreatedAt(saved.getCreatedAt());
                response.setTitle(EncryptionUtil.decrypt(saved.getTitle()));
                response.setContent(EncryptionUtil.decrypt(saved.getContent()));
                return response;
            } else {
                // Just return decrypted existing note
                return decryptNote(existing);
            }
        } else {
            // Create new note
            logger.info("No duplicate found, creating new note for user {}", userId);
            return saveNote(note);
        }
    }

    @Transactional(readOnly = true)
    public Optional<Note> getNoteByTitle(String title) {
        if (title == null || title.isBlank()) {
            throw new IllegalArgumentException("Cannot search for a blank note");
        }
        return noteRepository.findByTitleIgnoreCase(title)
                .map(this::decryptNote);
    }

    public Optional<Note> getNoteByCretedAt(LocalDateTime createdAt) {
        return noteRepository.findByCreatedAt(createdAt);
    }

    @Transactional(readOnly = true)
    public List<Note> getNotesByUserId(Long userId) {
        List<Note> notes = noteRepository.findByUserId(userId);
        return notes.stream().map(this::decryptNote).toList();
    }

    // Returns true if the note existed and was deleted, false if it didn't exist
    @Transactional
    public boolean deleteNoteById(Long id) {
        if (noteRepository.existsById(id)) {
            noteRepository.deleteById(id);
            return true;
        }
        return false;
    }

    @Transactional
    public Note updateNote(Note note) {
        // Encrypt title and content before updating
        if (note.getTitle() != null) {
            note.setTitle(EncryptionUtil.encrypt(note.getTitle()));
        }
        if (note.getContent() != null) {
            note.setContent(EncryptionUtil.encrypt(note.getContent()));
        }
        Note saved = noteRepository.save(note);
        
        // Return a copy with decrypted content
        Note response = new Note();
        response.setId(saved.getId());
        response.setUserId(saved.getUserId());
        response.setCreatedAt(saved.getCreatedAt());
        
        if (saved.getTitle() != null) {
            response.setTitle(EncryptionUtil.decrypt(saved.getTitle()));
        }
        if (saved.getContent() != null) {
            response.setContent(EncryptionUtil.decrypt(saved.getContent()));
        }
        return response;
    }

    // New: Delete a note by its title (case-insensitive). Returns true if deleted.
    @Transactional
    public boolean deleteNoteByTitle(String title) {
        if (title == null || title.isBlank()) return false;
        Optional<Note> existing = noteRepository.findByTitleIgnoreCase(title);
        if (existing.isPresent()) {
            noteRepository.delete(existing.get());
            return true;
        }
        return false;
    }

    // New: Update a note found by title (case-insensitive). Returns the updated note with decrypted content.
    @Transactional
    public Optional<Note> updateNoteByTitle(String title, Note noteDetails) {
        if (title == null || title.isBlank()) return Optional.empty();
        Optional<Note> existingOpt = noteRepository.findByTitleIgnoreCase(title);
        if (existingOpt.isPresent()) {
            Note noteToUpdate = existingOpt.get();
            // Update allowed fields (encrypt before saving)
            if (noteDetails.getTitle() != null && !noteDetails.getTitle().isBlank()) {
                noteToUpdate.setTitle(EncryptionUtil.encrypt(noteDetails.getTitle()));
            }
            if (noteDetails.getContent() != null) {
                noteToUpdate.setContent(EncryptionUtil.encrypt(noteDetails.getContent()));
            }
            Note saved = noteRepository.save(noteToUpdate);
            
            // Return a copy with decrypted content
            Note response = new Note();
            response.setId(saved.getId());
            response.setUserId(saved.getUserId());
            response.setCreatedAt(saved.getCreatedAt());
            response.setTitle(EncryptionUtil.decrypt(saved.getTitle()));
            response.setContent(EncryptionUtil.decrypt(saved.getContent()));
            
            return Optional.of(response);
        }
        return Optional.empty();
    }

    // New: Search notes that contain the title (case-insensitive) and return with decrypted content
    @Transactional(readOnly = true)
    public List<Note> searchNotesByTitleAndUserId(String title, Long userId) {
        if (title == null) title = "";
        List<Note> notes = noteRepository.findByTitleIgnoreCaseContainingAndUserId(title, userId);
        return notes.stream().map(this::decryptNote).toList();
    }

    @Transactional(readOnly = true)
    public Optional<Note> getNoteByTitleAndUserId(String title, Long userId) {
        if (title == null || title.isBlank()) {
            throw new IllegalArgumentException("Cannot search for a blank note");
        }
        return noteRepository.findByTitleIgnoreCaseAndUserId(title, userId)
                .map(this::decryptNote);
    }

    @Transactional
    public Optional<Note> updateNoteByTitleAndUserId(String title, Note noteDetails, Long userId) {
        if (title == null || title.isBlank()) return Optional.empty();
        Optional<Note> existingOpt = noteRepository.findByTitleIgnoreCaseAndUserId(title, userId);
        if (existingOpt.isPresent()) {
            Note noteToUpdate = existingOpt.get();
            // Update allowed fields (encrypt before saving)
            if (noteDetails.getTitle() != null && !noteDetails.getTitle().isBlank()) {
                noteToUpdate.setTitle(EncryptionUtil.encrypt(noteDetails.getTitle()));
            }
            if (noteDetails.getContent() != null) {
                noteToUpdate.setContent(EncryptionUtil.encrypt(noteDetails.getContent()));
            }
            noteToUpdate.setUserId(userId); // Ensure userId remains
            Note saved = noteRepository.save(noteToUpdate);
            
            // Return a copy with decrypted title and content
            Note response = new Note();
            response.setId(saved.getId());
            response.setUserId(saved.getUserId());
            response.setCreatedAt(saved.getCreatedAt());
            response.setTitle(EncryptionUtil.decrypt(saved.getTitle()));
            response.setContent(EncryptionUtil.decrypt(saved.getContent()));
            
            return Optional.of(response);
        }
        return Optional.empty();
    }

    @Transactional
    public boolean deleteNoteByTitleAndUserId(String title, Long userId) {
        if (title == null || title.isBlank()) return false;
        Optional<Note> existing = noteRepository.findByTitleIgnoreCaseAndUserId(title, userId);
        if (existing.isPresent()) {
            noteRepository.delete(existing.get());
            return true;
        }
        return false;
    }

    private Note decryptNote(Note note) {
        Note copy = new Note();
        copy.setId(note.getId());
        copy.setUserId(note.getUserId());
        copy.setCreatedAt(note.getCreatedAt());
        copy.setTitle(EncryptionUtil.decrypt(note.getTitle()));
        copy.setContent(EncryptionUtil.decrypt(note.getContent()));
        return copy;
    }
}
