@refresh-chapters
Feature: Refresh Library Manga Chapters

  As a user, I want to refresh the chapter list for a manga in my library
  so that new chapters from the content provider are synced into my local database.

  Background:
    Given a seeded library with "Alpha Manga" is running
    And I open the library

  @smoke
  Scenario: Refresh chapter list when new chapters are available
    When I click on the manga "Alpha Manga" in the library
    Then I am on the library manga details page for "Alpha Manga"
    And the manga has "3" chapters
    When I click the refresh button
    Then the chapter list updates
    And new chapters are added to the list
    And I am notified about the new chapters

  Scenario: Refresh chapter list when already up to date
    When I click on the manga "Alpha Manga" in the library
    Then I am on the library manga details page for "Alpha Manga"
    And the manga is up to date with the provider
    When I click the refresh button
    Then the chapter list updates
    And I see a message "Up to date"
    And the chapter list is unchanged
