# frozen_string_literal: true
#
# Pure-Ruby usage of the Money class, as provided by go-embedded-ruby (rbgo).
# Run it with:  rbgo examples/money_usage.rb

require "money"

# Amounts are held in the currency's smallest unit (here, cents).
price = Money.new(1099, "EUR")   # 10,99 €
tax   = Money.new(220, "EUR")    #  2,20 €

# Localised formatting and plain string form.
puts price.format
puts price.to_s

# Arithmetic stays in the currency.
total = price + tax
puts total.format
