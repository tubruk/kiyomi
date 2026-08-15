@library-manga-details
Feature: Library Manga Details

  As a user, I want to view detailed information about a manga in my library,
  including its cover image, title, aliases, author/artist credits, tags,
  synopsis, reading status, and chapter list.

  Background:
    Given a seeded library with manga-x is running
    And I open the library

  @smoke
  Scenario: Inspect read-only library manga details
    When I click on the manga "Alpha Manga" in the library
    Then I am on the library manga details page for "Alpha Manga"
    And I see the manga cover image
    And I see the manga title "Alpha Manga"
    And I see the manga aliases containing "Alpha Alternate Title"
    And I see the merged author and artist "Alpha Author"
    And I see the manga tags including "Action"
    And I see the manga synopsis containing "Synopsis for Alpha Manga"
    And I see the manga reading status is "Reading"
    And the manga has "5" chapters listed
