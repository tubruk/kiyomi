Feature: Reading Progress Tracking and Resume Journey
  As a Kiyomi manga reader
  I want my reading progress and chapter completion state to be saved automatically
  So that I can resume reading seamlessly across sessions and track unread chapters in my library.

  Background:
    Given the Kiyomi server and library storage are initialized

  @reader @progress
  Scenario: Resume reading an in-progress chapter
    Given I have a manga "Chainsaw Man" in my library with 10 chapters
    And I previously stopped reading Chapter 2 at page 12
    When I view the manga details page for "Chainsaw Man"
    Then the primary reading button should display "Resume Ch. 2 (p. 12)"
    When I click the primary reading button
    Then the reader should load Chapter 2
    And the reader view should automatically scroll to page 12

  @reader @progress @autosync
  Scenario: Automatically save page progress while reading
    Given I am reading Chapter 3 (20 pages total) of "Chainsaw Man"
    When I scroll down to page 15 and pause reading for 1.5 seconds
    Then a progress update request should be sent with "last_read_page" set to 15
    And the manga "last_read_at" timestamp should be updated

  @reader @progress @completion
  Scenario: Automatically mark chapter as completed on reaching the last page
    Given I am reading Chapter 3 (20 pages total) of "Chainsaw Man"
    When I scroll to page 20
    Then the chapter status "is_read" should automatically be updated to true
    And the chapter list item for Chapter 3 should display a completed checkmark

  @reader @progress @navigation
  Scenario: Automatically mark chapter as completed when clicking Next Chapter
    Given I am reading Chapter 4 at page 10 of 25
    When I click the "Next Chapter" button in the reader toolbar
    Then Chapter 4 should be marked with "is_read" set to true
    And the reader should load Chapter 5 at page 1

  @library @progress @badges
  Scenario: Display unread chapter count badges on library cards
    Given I have a manga "One Piece" in my library with 100 total chapters
    And 20 chapters have "is_read" set to true
    When I navigate to the Library Shelf page
    Then the manga card for "One Piece" should display an "80 unread" badge

  @library @progress @badges
  Scenario: Display completed badge when all chapters are read
    Given I have a manga "Mob Psycho 100" in my library with 10 total chapters
    And all 10 chapters have "is_read" set to true
    When I navigate to the Library Shelf page
    Then the manga card for "Mob Psycho 100" should display a "Completed" badge
