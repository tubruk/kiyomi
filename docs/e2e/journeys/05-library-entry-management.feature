@library-entry-management
Feature: Library Entry Management

  As a user, I want to manage my library entries by updating reading status,
  toggling favorites, setting personal ratings, and editing personal notes,
  and have these changes persist across page navigations.

  Background:
    Given a seeded library with manga-x is running
    And I open the library

  Scenario: Update reading status and persist across navigation
    When I click on the manga "Alpha Manga" in the library
    Then I am on the library manga details page for "Alpha Manga"
    And I see the manga reading status is "Reading"
    When I select the reading status "Completed"
    Then I see the manga reading status is "Completed"
    When I click "Back to Library"
    Then I am on the library page
    When I click on the manga "Alpha Manga" in the library
    Then I am on the library manga details page for "Alpha Manga"
    And I see the manga reading status is "Completed"

  Scenario: Toggle favorite status and persist across navigation
    When I click on the manga "Alpha Manga" in the library
    Then I am on the library manga details page for "Alpha Manga"
    And the manga is not marked as favorite
    When I toggle the favorite button
    Then the manga is marked as favorite
    When I click "Back to Library"
    Then I am on the library page
    When I click on the manga "Alpha Manga" in the library
    Then I am on the library manga details page for "Alpha Manga"
    And the manga is marked as favorite

  Scenario: Set personal rating and persist across navigation
    When I click on the manga "Alpha Manga" in the library
    Then I am on the library manga details page for "Alpha Manga"
    And I see the manga rating is "Unrated"
    When I set the rating to "4" stars
    Then I see the manga rating is "8/10"
    When I click "Back to Library"
    Then I am on the library page
    When I click on the manga "Alpha Manga" in the library
    Then I am on the library manga details page for "Alpha Manga"
    And I see the manga rating is "8/10"

  Scenario: Add personal notes and persist across navigation
    When I click on the manga "Alpha Manga" in the library
    Then I am on the library manga details page for "Alpha Manga"
    And I see no personal notes added
    When I set the personal notes to "Great world building and character pacing."
    Then I see the manga personal notes contain "Great world building and character pacing."
    When I click "Back to Library"
    Then I am on the library page
    When I click on the manga "Alpha Manga" in the library
    Then I am on the library manga details page for "Alpha Manga"
    And I see the manga personal notes contain "Great world building and character pacing."

  @smoke
  Scenario: Manage multiple entry fields and verify all persist across navigation
    When I click on the manga "Alpha Manga" in the library
    Then I am on the library manga details page for "Alpha Manga"
    When I select the reading status "Plan to Read"
    And I toggle the favorite button
    And I set the rating to "5" stars
    And I set the personal notes to "Must read after finishing current series."
    Then I see the manga reading status is "Plan to Read"
    And the manga is marked as favorite
    And I see the manga rating is "10/10"
    And I see the manga personal notes contain "Must read after finishing current series."
    When I click "Back to Library"
    Then I am on the library page
    When I click on the manga "Alpha Manga" in the library
    Then I am on the library manga details page for "Alpha Manga"
    And I see the manga reading status is "Plan to Read"
    And the manga is marked as favorite
    And I see the manga rating is "10/10"
    And I see the manga personal notes contain "Must read after finishing current series."
