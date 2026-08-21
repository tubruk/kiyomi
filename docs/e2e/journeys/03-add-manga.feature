@add-manga
Feature: Add Manga to Library

  As a user, I want to add manga to my library so I can track and read them.

  Background:
    Given a clean server with empty library is running
    And I open the library

  @smoke
  Scenario: Happy path — add manga from empty library via Explore
    Given the library is empty
    When I click "Start Exploring"
    Then I am redirected to the Explore view
    When I search for "Alpha Manga"
    And I click the first result
    And I click "Add to Library"
    Then the manga "Alpha Manga" appears in the library
    And the manga cover thumbnail displays the content provider badge "mock"
    And the manga has "5" chapters listed
    When I click on the manga "Alpha Manga" in the library
    Then I am on the library manga details page for "Alpha Manga"
    And the provider section shows the provider "mock"

  Scenario: Manga already in library — duplicate prevented in Explore
    Given the manga "Alpha Manga" is already in the library
    When I open the Explore view
    And I search for "Alpha Manga"
    And I click the first result
    Then the "Add to Library" button is disabled or indicates "In Library"

  Scenario: Provider returns zero chapters — error surfaced on preview
    Given the mock provider has a manga with zero chapters
    When I open the Explore view
    And I search for "Empty Manga"
    And I click the first result
    Then an error is shown indicating no chapters were found
    And the "Add to Library" button is disabled
