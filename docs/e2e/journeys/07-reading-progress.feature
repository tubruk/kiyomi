@reading-progress
Feature: Reading Progress Tracking and Resume Journey

  As a user, I want my reading progress and chapter completion state to be saved automatically,
  so that I can resume reading seamlessly across sessions and see unread counters in my library.

  Background:
    Given a seeded library with manga-x is running
    And I open the library

  @smoke
  Scenario: Resume reading an in-progress chapter
    When I click on the manga "Alpha Manga" in the library
    Then I am on the library manga details page for "Alpha Manga"
    And I see the primary reading button displays "Start Reading"
    When I click "Start Reading"
    Then I am in the reader for "Alpha Manga"
    When I scroll down the reader view
    And I wait for progress to auto-sync
    When I click the back button in the reader
    Then I am on the library manga details page for "Alpha Manga"
    And I see the primary reading button displays "Resume"

  Scenario: Display unread badge on library manga cards
    Then I see the manga card for "Alpha Manga" displays unread chapter badge

  Scenario: Mark chapter as completed when reaching the end
    When I click on the manga "Alpha Manga" in the library
    And I click "Start Reading"
    Then I am in the reader for "Alpha Manga"
    When I scroll to the bottom of the chapter
    Then the current chapter is marked as read
