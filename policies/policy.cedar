@id("allow-create-book")
permit (
  principal,
  action == Action::"CreateBook",
  resource
);

@id("allow-get-published-book")
permit (
  principal,
  action == Action::"GetBook",
  resource
)
when { resource.status == "published" };

@id("allow-get-own-draft")
permit (
  principal,
  action == Action::"GetBook",
  resource
)
when { resource.status == "draft" && resource.author == principal.id };

@id("allow-list-books")
permit (
  principal,
  action == Action::"ListBooks",
  resource
);

@id("allow-update-own-book")
permit (
  principal,
  action == Action::"UpdateBook",
  resource
)
when { resource.author == principal.id };

@id("allow-delete-own-book")
permit (
  principal,
  action == Action::"DeleteBook",
  resource
)
when { resource.author == principal.id };
