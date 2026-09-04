# grubby
grubby is a recipe storage and sharing platform designed to aid in dinner decisions, by keeping recipes clear, concise and to the point. With dietary tags, difficulty ratings, spice rating and more for filtering recipes to your unique tastes.

## Motivation
My family is typically very busy, and dinner decisions typically come down to a back and forth conversation of "what do you want" "i don't know what do you want". And when trying to show anyone a recipe online, an ad typically popped up as I turned my phone or laptop around to show my suggestion. This was tedious especially because finding a recipe usually takes you to one of several hundreds if not thousands of recipe blog sites riddled with advertisements, and newsletters. Well grubby aims to solve this by creating a centralized place to store recipes, create cookbooks ( a collection of recipes that you can share with others on grubby ). Blend taste, spice, or even dietary preferences with other users to find a recipe that maybe you and others can agree on. As well as API integrations with nutritional databases to calculate macros in the recipe so you have a better idea of what you're eating. Without the long droning backstory of a recipe blog.


## Quick Start
grubby is designed to be deployed with docker and docker compose, so all you'll need is docker compose, and the ability to run a few containers.
Work in Progress... Check back later.

## Usage
grubby comes with several api endpoints you can use the frontend built-with grubby, or you can use the endpoints to make your own if you choose.

User Creation:
```JSON
POST /api/users
{
"email":"test123@gmail.com",
"display_name":"testguy123",
"password":"easy123"
}

expected response:
{
"id":"7bc06614-8149-4fbc-b07d-9a2d300d5fec",
"created_at":"2026-09-04T12:02:55.11709-06:00",
"updated_at":"2026-09-04T12:02:55.11709-06:00",
"email":"test123@gmail.com",
"display_name":"testguy123",
"is_premium":false
}
```
The above is the account creation endpoint, and some filler data as an example. the input and output has been pretty printed for readability.

## Contributing
This is a personal project and at this time, I won't be accepting pull requests. However feel free to fork the repo and make grubby your own.
