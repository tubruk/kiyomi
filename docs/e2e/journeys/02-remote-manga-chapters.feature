@remote-manga-chapters
Feature: Remote Manga Chapter Browsing and Direct Reading

  As a user, I want to browse chapter lists for remote manga from a provider and read remote chapters directly before adding the manga to my library.

  Background:
    Given a clean server with empty library is running

  Scenario: Browse remote manga chapters from provider catalog
    Given I open the Explore view
    When I select the provider "Mock Provider"
    And I click the manga "Alpha Manga"
    Then I am on the remote manga details page for "Alpha Manga"
    And I see the manga details including synopsis and chapter list
    And the manga has "5" chapters listed

  @smoke
  Scenario: Read remote chapter before adding to library
    Given I open the Explore view
    When I select the provider "Mock Provider"
    And I click the manga "Alpha Manga"
    Then I am on the remote manga details page for "Alpha Manga"
    When I click on chapter "1"
    Then the reader opens
    And I am on page 1 of chapter 1

  Scenario: Add remote manga to library after inspecting chapters
    Given I open the Explore view
    When I select the provider "Mock Provider"
    And I click the manga "Alpha Manga"
    Then I am on the remote manga details page for "Alpha Manga"
    When I click "Add to Library"
    Then the manga "Alpha Manga" appears in the library
    And the manga has "5" chapters listed

  Scenario: View remote manga explicitly marked as unavailable upstream
    Given the provider has a manga "Unavailable Manga" explicitly marked as unavailable
    When I open the Explore view
    And I select the provider "Mock Provider"
    Then the manga card for "Unavailable Manga" displays an "Unavailable" badge
    When I click the manga "Unavailable Manga"
    Then I am on the remote manga details page for "Unavailable Manga"
    And I see the manga details including synopsis and metadata
    And the chapter list section displays "Content Unavailable in Mock Provider"

